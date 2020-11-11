package apis

import (
	kafkasourcev1beta1 "knative.dev/eventing-contrib/kafka/source/pkg/apis/sources/v1beta1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Add Knative Eventing Kafka source scheme used in Telemetry
	AddToSchemes = append(AddToSchemes, kafkasourcev1beta1.AddToScheme)
}
