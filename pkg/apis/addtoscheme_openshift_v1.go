package apis

import (
	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Adds schema for openshift apis
	AddToSchemes = append(AddToSchemes, configv1.Install, routev1.Install, maistrav1.SchemeBuilder.AddToScheme)
}
