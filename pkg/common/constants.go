package common

const (
	ServingDownstreamDomain  = "serving.knative.openshift.io"
	EventingDownstreamDomain = "eventing.knative.openshift.io"

	// Label keys being used to tag the owned resources by instance
	ServingOwnerName       = ServingDownstreamDomain + "/ownerName"
	ServingOwnerNamespace  = ServingDownstreamDomain + "/ownerNamespace"
	EventingOwnerName      = EventingDownstreamDomain + "/ownerName"
	EventingOwnerNamespace = EventingDownstreamDomain + "/ownerNamespace"
)
