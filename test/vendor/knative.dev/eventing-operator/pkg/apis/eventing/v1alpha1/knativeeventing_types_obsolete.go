/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Eventing is the Schema for the eventings API
// +k8s:openapi-gen=true
type Eventing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventingSpec   `json:"spec,omitempty"`
	Status EventingStatus `json:"status,omitempty"`
}

// EventingSpec defines the desired state of Eventing
// +k8s:openapi-gen=true
type EventingSpec struct {
}

// EventingStatus defines the observed state of Eventing
// +k8s:openapi-gen=true
type EventingStatus struct {
	duckv1beta1.Status `json:",inline"`

	// The version of the installed release
	// +optional
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventingList contains a list of Eventing
type EventingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Eventing `json:"items"`
}
