package common

const (
	DownstreamDomain         = "knative.openshift.io"
	ServingDownstreamDomain  = "serving.knative.openshift.io"
	EventingDownstreamDomain = "eventing.knative.openshift.io"

	// Label keys being used to tag the owned resources by instance
	ServingOwnerName       = ServingDownstreamDomain + "/ownerName"
	ServingOwnerNamespace  = ServingDownstreamDomain + "/ownerNamespace"
	EventingOwnerName      = EventingDownstreamDomain + "/ownerName"
	EventingOwnerNamespace = EventingDownstreamDomain + "/ownerNamespace"

	ServerlessCommonLabelKey   = DownstreamDomain + "/part-of"
	ServerlessCommonLabelValue = "openshift-serverless"
)
