package common

// Annotation keys being used to tag the owned resources by instance
const (
	ServingOwnerName                 = "serving.knative.openshift.io/ownerName"
	ServingOwnerNamespace            = "serving.knative.openshift.io/ownerNamespace"
	EventingOwnerName                = "eventing.knative.openshift.io/ownerName"
	EventingOwnerNamespace           = "eventing.knative.openshift.io/ownerNamespace"
	ServerlessOperatorOwnerName      = "operator.knative.openshift.io/ownerName"
	ServerlessOperatorOwnerNamespace = "operator.knative.openshift.io/ownerNamespace"
	KafkaOwnerName                   = "knativekafkas.operator.serverless.openshift.io/ownerName"
	KafkaOwnerNamespace              = "knativekafkas.operator.serverless.openshift.io/ownerNamespace"

	// The namespace of the pod will be available through this key.
	NamespaceEnvKey = "NAMESPACE"
)
