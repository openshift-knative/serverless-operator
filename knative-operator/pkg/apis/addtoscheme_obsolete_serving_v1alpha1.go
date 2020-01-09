package apis

import (
	"github.com/openshift-knative/serverless-operator/serving/operator/pkg/apis/serving/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Adds schema for knative serving
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
}
