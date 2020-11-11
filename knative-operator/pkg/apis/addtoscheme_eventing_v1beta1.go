package apis

import (
	eventingsourcesv1beta1 "knative.dev/eventing/pkg/apis/sources/v1beta1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Add Knative Eventing source scheme used in Telemetry
	AddToSchemes = append(AddToSchemes, eventingsourcesv1beta1.AddToScheme)
}
