package v1alpha1

import (
	"github.com/knative/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	DependenciesInstalled apis.ConditionType = "DependenciesInstalled"
	InstallSucceeded      apis.ConditionType = "InstallSucceeded"
	DeploymentsAvailable  apis.ConditionType = "DeploymentsAvailable"
)

// KnativeServingSpec defines the desired state of KnativeServing
// +k8s:openapi-gen=true
type KnativeServingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// A means to override the corresponding entries in the upstream configmaps
	// +optional
	Config map[string]map[string]string `json:"config,omitempty"`
}

// KnativeServingStatus defines the observed state of KnativeServing
// +k8s:openapi-gen=true
type KnativeServingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags:
	// https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// The version of the installed release
	// +optional
	Version string `json:"version,omitempty"`
	// The latest available observations of a resource's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions apis.Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KnativeServing is the Schema for the knativeservings API
// +k8s:openapi-gen=true
type KnativeServing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnativeServingSpec   `json:"spec,omitempty"`
	Status KnativeServingStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KnativeServingList contains a list of KnativeServing
type KnativeServingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnativeServing `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KnativeServing{}, &KnativeServingList{})
}
