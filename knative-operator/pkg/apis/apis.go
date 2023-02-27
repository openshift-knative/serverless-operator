package apis

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Adds schema for core k8s types.
	AddToSchemes = append(AddToSchemes, scheme.AddToScheme)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
