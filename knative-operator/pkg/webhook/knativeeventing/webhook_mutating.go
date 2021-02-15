package knativeeventing

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Configurator annotates KEs
type Configurator struct {
	Decoder *admission.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*Configurator)(nil)

// Configurator adds an annotation to every incoming
// KnativeEventing CR.
func (v *Configurator) Handle(ctx context.Context, req admission.Request) admission.Response {
	ke := &eventingv1alpha1.KnativeEventing{}

	err := v.Decoder.Decode(req, ke)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	common.MutateEventing(ke)

	marshaled, err := json.Marshal(ke)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshaled)
}
