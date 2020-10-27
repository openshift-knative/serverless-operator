package testutil

import (
	"encoding/json"

	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// RequestFor generates an admission request for the given object.
func RequestFor(obj runtime.Object) (types.Request, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return types.Request{}, err
	}
	return types.Request{
		AdmissionRequest: &v1beta1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw:    b,
				Object: obj,
			},
		},
	}, nil
}
