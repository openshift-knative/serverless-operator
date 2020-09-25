package apis

import (
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Adds schema for knative serving
	AddToSchemes = append(AddToSchemes, v1alpha1.AddToScheme)
}
