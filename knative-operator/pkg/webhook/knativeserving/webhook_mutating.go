package knativeserving

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/appscode/jsonpatch"
	"github.com/openshift-knative/knative-serving-openshift/pkg/common"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"

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

// Add creates a new KnativeServing Webhook
func MutatingWebhook(mgr manager.Manager) (webhook.Webhook, error) {
	common.Log.Info("Setting up mutating webhook for KnativeServing")
	return builder.NewWebhookBuilder().
		Name("mutating.knativeserving.openshift.io").
		Mutating().
		Operations(admissionregistrationv1beta1.Create).
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
	return PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshaled)
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

// PatchResponseFromRaw takes 2 byte arrays and returns a new response with json patch.
// The original object should be passed in as raw bytes to avoid the roundtripping problem
// described in https://github.com/kubernetes-sigs/kubebuilder/issues/510.
func PatchResponseFromRaw(original, current []byte) types.Response {
	patches, err := jsonpatch.CreatePatch(original, current)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return types.Response{
		Patches: patches,
		Response: &admissionv1beta1.AdmissionResponse{
			Allowed:   true,
			PatchType: func() *admissionv1beta1.PatchType { pt := admissionv1beta1.PatchTypeJSONPatch; return &pt }(),
		},
	}
}
