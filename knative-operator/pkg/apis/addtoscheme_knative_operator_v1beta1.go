package apis

import (
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Adds schema for knative serving
	AddToSchemes = append(AddToSchemes, operatorv1beta1.AddToScheme)
}
