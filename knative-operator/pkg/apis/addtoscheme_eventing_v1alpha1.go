package apis

import (
	"knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Adds schema for knative eventing
	AddToSchemes = append(AddToSchemes, v1alpha1.AddToScheme)
}
