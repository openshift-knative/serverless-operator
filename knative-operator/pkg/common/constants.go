package common

const (
	OperatorDownstreamDomain = "operator.knative.openshift.io"
	KafkaDownstreamDomain    = "knativekafkas.operator.serverless.openshift.io"

	// Label keys being used to tag the owned resources by instance
	ServerlessOperatorOwnerName      = OperatorDownstreamDomain + "/ownerName"
	ServerlessOperatorOwnerNamespace = OperatorDownstreamDomain + "/ownerNamespace"
	KafkaOwnerName                   = KafkaDownstreamDomain + "/ownerName"
	KafkaOwnerNamespace              = KafkaDownstreamDomain + "/ownerNamespace"

	VolumeChecksumAnnotation = OperatorDownstreamDomain + "/configmap-volume-checksum"

	// The namespace of the pod will be available through this key.
	NamespaceEnvKey = "NAMESPACE"
)
