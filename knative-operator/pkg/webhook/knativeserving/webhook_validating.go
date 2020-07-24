package knativeserving

import (
	"context"
	"net/http"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
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
		Operations(admissionregistrationv1beta1.Create).
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

	allowed, reason, err := common.Validate(ctx, v.client, ks)

	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
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
