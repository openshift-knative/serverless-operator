package apis

import (
	configv1 "github.com/openshift/api/config/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Adds schema for openshift config
	AddToSchemes = append(AddToSchemes, configv1.Install)
}
