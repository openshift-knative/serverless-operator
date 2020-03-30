package knativeeventing

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	webhookutil "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/util"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	eventingv1alpha1 "knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Creates a new validating KnativeEventing Webhook
func ValidatingWebhook(mgr manager.Manager) (webhook.Webhook, error) {
	common.Log.Info("Setting up validating webhook for KnativeEventing")
	return builder.NewWebhookBuilder().
		Name("validating.knativeeventing.openshift.io").
		Validating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(mgr).
		ForType(&eventingv1alpha1.KnativeEventing{}).
		Handlers(&KnativeEventingValidator{}).
		Build()
}

// KnativeEventingValidator validates KnativeEventing CR's
type KnativeEventingValidator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*KnativeEventingValidator)(nil)

// What makes us a webhook
func (v *KnativeEventingValidator) Handle(ctx context.Context, req types.Request) types.Response {
	ke := &eventingv1alpha1.KnativeEventing{}

	err := v.decoder.Decode(req, ke)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validate(ctx, ke)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

// KnativeEventingValidator checks for a minimum OpenShift version
func (v *KnativeEventingValidator) validate(ctx context.Context, ke *eventingv1alpha1.KnativeEventing) (allowed bool, reason string, err error) {
	log := common.Log.WithName("validate")
	stages := []func(context.Context, *eventingv1alpha1.KnativeEventing) (bool, string, error){
		v.validateNamespace,
		v.validateVersion,
		v.validateLoneliness,
	}
	for _, stage := range stages {
		allowed, reason, err = stage(ctx, ke)
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

// KnativeEventingValidator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = (*KnativeEventingValidator)(nil)

// InjectClient injects the client.
func (v *KnativeEventingValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// KnativeEventingValidator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = (*KnativeEventingValidator)(nil)

// InjectDecoder injects the decoder.
func (v *KnativeEventingValidator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}

// validate minimum openshift version
func (v *KnativeEventingValidator) validateVersion(ctx context.Context, _ *eventingv1alpha1.KnativeEventing) (bool, string, error) {
	return webhookutil.ValidateOpenShiftVersion(ctx, v.client)
}

// validate required namespace, if any
func (v *KnativeEventingValidator) validateNamespace(ctx context.Context, ke *eventingv1alpha1.KnativeEventing) (bool, string, error) {
	ns, required := os.LookupEnv("REQUIRED_EVENTING_NAMESPACE")
	if required && ns != ke.Namespace {
		return false, fmt.Sprintf("KnativeEventing may only be created in %s namespace", ns), nil
	}
	return true, "", nil
}

// validate this is the only KE in this namespace
func (v *KnativeEventingValidator) validateLoneliness(ctx context.Context, ke *eventingv1alpha1.KnativeEventing) (bool, string, error) {
	list := &eventingv1alpha1.KnativeEventingList{}
	if err := v.client.List(ctx, &client.ListOptions{Namespace: ke.Namespace}, list); err != nil {
		return false, "Unable to list KnativeEventings", err
	}
	for _, v := range list.Items {
		if ke.Name != v.Name {
			return false, "Only one KnativeEventing allowed per namespace", nil
		}
	}
	return true, "", nil
}
