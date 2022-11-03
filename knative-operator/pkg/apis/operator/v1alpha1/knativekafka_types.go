package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/base"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// KnativeKafkaSpec defines the desired state of KnativeKafka
// +k8s:openapi-gen=true

const DefaultLogLevel = "INFO"

type KnativeKafkaSpec struct {
	// Allows configuration for KafkaBroker installation
	// +optional
	Broker Broker `json:"broker,omitempty"`
	// Allows configuration for KafkaSource installation
	// +optional
	Source Source `json:"source,omitempty"`

	// Allows configuration for KafkaSink installation
	// +optional
	Sink Sink `json:"sink,omitempty"`
	// Allows configuration for KafkaChannel installation
	// +optional
	Channel Channel `json:"channel,omitempty"`

	// A means to override the corresponding entries in the upstream configmaps
	// +optional
	Config base.ConfigMapData `json:"config,omitempty"`

	// HighAvailability allows specification of HA control plane.
	// +optional
	HighAvailability *base.HighAvailability `json:"high-availability,omitempty"`

	// Set logging configuration of the data plane.
	// +optional
	Logging *Logging `json:"logging,omitempty"`

	// Workloads overrides workloads configurations such as resources and replicas.
	// +optional
	Workloads []base.WorkloadOverride `json:"workloads,omitempty"`
}

// KnativeKafkaStatus defines the observed state of KnativeKafka
// +k8s:openapi-gen=true
type KnativeKafkaStatus struct {
	duckv1.Status `json:",inline"`

	// The version of the installed release
	// +optional
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KnativeKafka is the Schema for the knativekafkas API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KnativeKafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnativeKafkaSpec   `json:"spec,omitempty"`
	Status KnativeKafkaStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KnativeKafkaList contains a list of KnativeKafka
type KnativeKafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnativeKafka `json:"items"`
}

type BrokerDefaultConfig struct {
	// BootstrapServers is the default comma-separated string of bootstrapservers that the
	// brokers will use, but can be overridden on the individual broker object's config map.
	// +optional
	BootstrapServers string `json:"bootstrapServers"`

	// NumPartitions is the number of partitions of a Kafka topic. By default, it is set to 10.
	NumPartitions int32 `json:"numPartitions"`

	// ReplicationFactor is the replication factor of a Kafka topic. By default, it is set to 3.
	ReplicationFactor int16 `json:"replicationFactor"`

	// AuthSecretName is the name of the secret that contains Kafka
	// auth configuration.
	// +optional
	AuthSecretName string `json:"authSecretName"`
}

// Broker allows configuration for KafkaBroker installation
type Broker struct {
	// Enabled defines if the KafkaBroker installation is enabled
	Enabled bool `json:"enabled"`

	// DefaultConfig settings for the Openshift cluster
	DefaultConfig BrokerDefaultConfig `json:"defaultConfig"`
}

// Source allows configuration for KafkaSource installation
type Source struct {
	// Enabled defines if the KafkaSource installation is enabled
	Enabled bool `json:"enabled"`
}

// Sink allows configuration for KafkaSink installation
type Sink struct {
	// Enabled defines if the KafkaSink installation is enabled
	Enabled bool `json:"enabled"`
}

// Channel allows configuration for KafkaSource installation
type Channel struct {
	// Enabled defines if the KafkaChannel installation is enabled
	Enabled bool `json:"enabled"`

	// BootstrapServers is comma separated string of bootstrapservers that the
	// KafkaChannels will use
	// +optional
	BootstrapServers string `json:"bootstrapServers"`

	// AuthSecretNamespace is the namespace of the secret that contains Kafka
	// auth configuration.
	// +optional
	AuthSecretNamespace string `json:"authSecretNamespace"`

	// AuthSecretName is the name of the secret that contains Kafka
	// auth configuration.
	// +optional
	AuthSecretName string `json:"authSecretName"`
}

type Logging struct {
	// Defines the log level. Allowed values are 'TRACE', 'DEBUG', 'INFO', 'WARN' and 'ERROR'.
	// The default value is 'INFO'.
	// +optional
	Level string `json:"level,omitempty"`
}

func init() {
	SchemeBuilder.Register(&KnativeKafka{}, &KnativeKafkaList{})
}
