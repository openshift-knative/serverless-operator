package common

// Annotation keys being used to tag the owned resources by instance
const (
	ServingOwnerName                 = "serving.knative.openshift.io/ownerName"
	ServingOwnerNamespace            = "serving.knative.openshift.io/ownerNamespace"
	ServerlessOperatorOwnerName      = "operator.knative.openshift.io/ownerName"
	ServerlessOperatorOwnerNamespace = "operator.knative.openshift.io/ownerNamespace"
	KafkaOwnerName                   = "kafka.knative.openshift.io/ownerName"
	KafkaOwnerNamespace              = "kafka.knative.openshift.io/ownerNamespace"
)
