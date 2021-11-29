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

package defaults

import (
	"context"

	"knative.dev/pkg/apis"
	servingv1alpha1 "knative.dev/serving/pkg/apis/serving/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetDomainMapping is a wrapper around Configuration.
type TargetDomainMapping struct {
	servingv1alpha1.DomainMapping `json:",inline"`
}

// Verify that Deployment adheres to the appropriate interfaces.
var (
	// Check that Deployment can be defaulted.
	_ apis.Defaultable = (*TargetDomainMapping)(nil)
	_ apis.Validatable = (*TargetDomainMapping)(nil)
)

// SetDefaults implements apis.Defaultable
func (r *TargetDomainMapping) SetDefaults(ctx context.Context) {
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
	r.Annotations[openshiftPassthrough] = "true"
}

// Validate returns nil due to no need for validation
func (r *TargetDomainMapping) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
