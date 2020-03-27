package knativeserving

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/util"
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

// MutatingWebhook creates a new KnativeServing mutating webhook
func MutatingWebhook(mgr manager.Manager) (webhook.Webhook, error) {
	common.Log.Info("Setting up mutating webhook for KnativeServing")
	return builder.NewWebhookBuilder().
		Name("mutating.knativeserving.openshift.io").
		Mutating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(mgr).
		ForType(&servingv1alpha1.KnativeServing{}).
		Handlers(&KnativeServingConfigurator{}).
		Build()
}

// KnativeServingConfigurator annotates Kss
type KnativeServingConfigurator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*KnativeServingConfigurator)(nil)

// KnativeServingConfigurator adds an annotation to every incoming
// KnativeServing CR.
func (a *KnativeServingConfigurator) Handle(ctx context.Context, req types.Request) types.Response {
	ks := &servingv1alpha1.KnativeServing{}

	err := a.decoder.Decode(req, ks)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	err = common.Mutate(ks, a.client)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}

	marshaled, err := json.Marshal(ks)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return util.PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshaled)
}

// KnativeServingConfigurator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = (*KnativeServingConfigurator)(nil)

// InjectClient injects the client.
func (v *KnativeServingConfigurator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// KnativeServingConfigurator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = (*KnativeServingConfigurator)(nil)

// InjectDecoder injects the decoder.
func (v *KnativeServingConfigurator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}
