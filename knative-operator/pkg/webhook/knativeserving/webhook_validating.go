package knativeserving

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	webhookutil "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/util"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Creates a new validating KnativeServing Webhook
func ValidatingWebhook(mgr manager.Manager) (webhook.Webhook, error) {
	common.Log.Info("Setting up validating webhook for KnativeServing")
	return builder.NewWebhookBuilder().
		Name("validating.knativeserving.openshift.io").
		Validating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(mgr).
		ForType(&servingv1alpha1.KnativeServing{}).
		Handlers(&KnativeServingValidator{}).
		Build()
}

// KnativeServingValidator validates KnativeServing CR's
type KnativeServingValidator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*KnativeServingValidator)(nil)

// What makes us a webhook
func (v *KnativeServingValidator) Handle(ctx context.Context, req types.Request) types.Response {
	ks := &servingv1alpha1.KnativeServing{}

	err := v.decoder.Decode(req, ks)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validate(ctx, ks)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

// KnativeServingValidator checks for a minimum OpenShift version
func (v *KnativeServingValidator) validate(ctx context.Context, ks *servingv1alpha1.KnativeServing) (allowed bool, reason string, err error) {
	log := common.Log.WithName("validate")
	stages := []func(context.Context, *servingv1alpha1.KnativeServing) (bool, string, error){
		v.validateNamespace,
		v.validateVersion,
		v.validateLoneliness,
	}
	for _, stage := range stages {
		allowed, reason, err = stage(ctx, ks)
		if len(reason) > 0 {
			if err != nil {
				log.Error(err, reason)
			} else {
				log.Info(reason)
			}
		}
		if !allowed {
			return
		}
	}
	return
}

// KnativeServingValidator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = (*KnativeServingValidator)(nil)

// InjectClient injects the client.
func (v *KnativeServingValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// KnativeServingValidator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = (*KnativeServingValidator)(nil)

// InjectDecoder injects the decoder.
func (v *KnativeServingValidator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}

// validate minimum openshift version
func (v *KnativeServingValidator) validateVersion(ctx context.Context, ks *servingv1alpha1.KnativeServing) (bool, string, error) {
	return webhookutil.ValidateOpenShiftVersion(ctx, v.client)
}

// validate required namespace, if any
func (v *KnativeServingValidator) validateNamespace(ctx context.Context, ks *servingv1alpha1.KnativeServing) (bool, string, error) {
	ns, required := os.LookupEnv("REQUIRED_SERVING_NAMESPACE")
	if required && ns != ks.Namespace {
		return false, fmt.Sprintf("KnativeServing may only be created in %s namespace", ns), nil
	}
	return true, "", nil
}

// validate this is the only KS in this namespace
func (v *KnativeServingValidator) validateLoneliness(ctx context.Context, ks *servingv1alpha1.KnativeServing) (bool, string, error) {
	list := &servingv1alpha1.KnativeServingList{}
	if err := v.client.List(ctx, &client.ListOptions{Namespace: ks.Namespace}, list); err != nil {
		return false, "Unable to list KnativeServings", err
	}
	for _, v := range list.Items {
		if ks.Name != v.Name {
			return false, "Only one KnativeServing allowed per namespace", nil
		}
	}
	return true, "", nil
}
