package knativeserving

import (
	"context"
	"encoding/json"
	"net/http"

	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Configurator annotates Kss
type Configurator struct {
	decoder *admission.Decoder
}

// NewConfigurator creates a new Configurator instance to configure KnativeServing CRs.
func NewConfigurator(decoder *admission.Decoder) *Configurator {
	return &Configurator{
		decoder: decoder,
	}
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*Configurator)(nil)

// Handle implements the Handler interface.
func (v *Configurator) Handle(ctx context.Context, req admission.Request) admission.Response {
	ks := &operatorv1beta1.KnativeServing{}

	err := v.decoder.Decode(req, ks)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Unset the entire registry section. We used to override it anyway, so there can't
	// be any userdata in there.
	// TODO: Remove in the 1.21 release to potentially make this usable for users.
	ks.Spec.CommonSpec.Registry.Override = nil

	defaultToKourier(ks)

	marshaled, err := json.Marshal(ks)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshaled)
}

// Technically this does nothing in terms of behavior (the default is assumed in the
// extension code of openshift-knative-operator already), but it fixes a UX nit where
// Kourier would be shown as enabled: false to the user if the ingress object is
// specified.
func defaultToKourier(ks *operatorv1beta1.KnativeServing) {
	if ks.Spec.Ingress == nil {
		return
	}

	if !ks.Spec.Ingress.Istio.Enabled && !ks.Spec.Ingress.Kourier.Enabled && !ks.Spec.Ingress.Contour.Enabled {
		ks.Spec.Ingress.Kourier.Enabled = true
	}
}
