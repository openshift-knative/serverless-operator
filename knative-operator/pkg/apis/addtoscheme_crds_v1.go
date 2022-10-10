package apis

import (
	apisextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Adds schema for k8s crds
	AddToSchemes = append(AddToSchemes, apisextensionv1.AddToScheme)
}
