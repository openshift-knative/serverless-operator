package knativeserving

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Configurator annotates Kss
type Configurator struct {
	client  client.Client
	decoder *admission.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*Configurator)(nil)

// Configurator adds an annotation to every incoming
// KnativeServing CR.
func (v *Configurator) Handle(ctx context.Context, req admission.Request) admission.Response {
	ks := &servingv1alpha1.KnativeServing{}

	err := v.decoder.Decode(req, ks)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	err = common.Mutate(ks, v.client)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	marshaled, err := json.Marshal(ks)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshaled)
}

// Configurator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = (*Configurator)(nil)

// InjectClient injects the client.
func (v *Configurator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// Configurator implements inject.Decoder.
// A decoder will be automatically injected.
var _ admission.DecoderInjector = (*Configurator)(nil)

// InjectDecoder injects the decoder.
func (v *Configurator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
