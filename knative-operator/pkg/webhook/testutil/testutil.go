package testutil

import (
	"encoding/json"

	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// RequestFor generates an admission request for the given object.
func RequestFor(obj runtime.Object) (admission.Request, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return admission.Request{}, err
	}
	return admission.Request{
		AdmissionRequest: v1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw:    b,
				Object: obj,
			},
		},
	}, nil
}
