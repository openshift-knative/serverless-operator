package apis

import (
	"github.com/maistra/istio-operator/pkg/apis/maistra/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Adds schema for maistra
	AddToSchemes = append(AddToSchemes, v1.SchemeBuilder.AddToScheme)
}
