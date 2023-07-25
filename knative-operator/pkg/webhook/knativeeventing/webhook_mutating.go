package knativeeventing

import (
	"context"
	"encoding/json"
	"net/http"

	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Configurator annotates KEs
type Configurator struct {
	decoder *admission.Decoder
}

// NewConfigurator creates a new Configurator instance to configure KnativeEventing CRs.
func NewConfigurator(decoder *admission.Decoder) *Configurator {
	return &Configurator{
		decoder: decoder,
	}
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*Configurator)(nil)

// Handle implements the Handler interface.
func (v *Configurator) Handle(_ context.Context, req admission.Request) admission.Response {
	ke := &operatorv1beta1.KnativeEventing{}

	err := v.decoder.Decode(req, ke)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Unset the entire registry section. We used to override it anyway, so there can't
	// be any userdata in there.
	// TODO: Remove in the 1.21 release to potentially make this usable for users.
	ke.Spec.CommonSpec.Registry.Override = nil

	marshaled, err := json.Marshal(ke)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshaled)
}
