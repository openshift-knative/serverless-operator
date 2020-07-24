package knativeeventing

import (
	"context"
	"net/http"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
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
		Operations(admissionregistrationv1beta1.Create).
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

	allowed, reason, err := common.Validate(ctx, v.client, ke)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
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
