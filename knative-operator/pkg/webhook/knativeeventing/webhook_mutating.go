package knativeeventing

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/util"
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

// MutatingWebhook creates a new KnativeEventing mutating webhook
func MutatingWebhook(mgr manager.Manager) (webhook.Webhook, error) {
	common.Log.Info("Setting up mutating webhook for KnativeEventing")
	return builder.NewWebhookBuilder().
		Name("mutating.knativeeventing.openshift.io").
		Mutating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(mgr).
		ForType(&eventingv1alpha1.KnativeEventing{}).
		Handlers(&KnativeEventingConfigurator{}).
		Build()
}

// KnativeEventingConfigurator annotates KEs
type KnativeEventingConfigurator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*KnativeEventingConfigurator)(nil)

// KnativeEventingConfigurator adds an annotation to every incoming
// KnativeEventing CR.
func (a *KnativeEventingConfigurator) Handle(ctx context.Context, req types.Request) types.Response {
	ke := &eventingv1alpha1.KnativeEventing{}

	err := a.decoder.Decode(req, ke)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	err = common.MutateEventing(ke, a.client)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}

	marshaled, err := json.Marshal(ke)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return util.PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshaled)
}

// KnativeEventingConfigurator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = (*KnativeEventingConfigurator)(nil)

// InjectClient injects the client.
func (v *KnativeEventingConfigurator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// KnativeEventingConfigurator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = (*KnativeEventingConfigurator)(nil)

// InjectDecoder injects the decoder.
func (v *KnativeEventingConfigurator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}
