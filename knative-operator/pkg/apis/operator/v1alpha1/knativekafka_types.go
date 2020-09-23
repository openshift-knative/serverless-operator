package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KnativeKafkaSpec defines the desired state of KnativeKafka
// +k8s:openapi-gen=true
type KnativeKafkaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Allows configuration for KafkaSource installation
	// +optional
	Source Source `json:"source,omitempty"`

	// Allows configuration for KafkaChannel installation
	// +optional
	Channel Channel `json:"channel,omitempty"`
}

// KnativeKafkaStatus defines the observed state of KnativeKafka
// +k8s:openapi-gen=true
type KnativeKafkaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

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

// Source allows configuration for KafkaSource installation
type Source struct {
	// Enabled defines if the KafkaSource installation is enabled
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

	// SetAsDefaultChannelProvisioner defines if KafkaChannel should be used
	// as the default channel implementation when using generic Channels.
	// +optional
	SetAsDefaultChannelProvisioner bool `json:"setAsDefaultChannelProvisioner,omitempty"`
}

func init() {
	SchemeBuilder.Register(&KnativeKafka{}, &KnativeKafkaList{})
}
