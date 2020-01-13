package apis

import (
	"knative.dev/pkg/apis/istio/v1alpha3"
)

func init() {
	AddToSchemes = append(AddToSchemes, v1alpha3.SchemeBuilder.AddToScheme)
}
