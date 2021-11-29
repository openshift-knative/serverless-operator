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
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetConfiguration is a wrapper around Configuration.
type TargetConfiguration struct {
	servingv1.Configuration `json:",inline"`
}

// Verify that Deployment adheres to the appropriate interfaces.
var (
	// Check that Deployment can be defaulted.
	_ apis.Defaultable = (*TargetConfiguration)(nil)
	_ apis.Validatable = (*TargetConfiguration)(nil)
)

// SetDefaults implements apis.Defaultable
func (r *TargetConfiguration) SetDefaults(ctx context.Context) {
	if r.Spec.Template.Annotations == nil {
		r.Spec.Template.Annotations = make(map[string]string)
	}

	r.Spec.Template.Annotations[sidecarInject] = "true"
	r.Spec.Template.Annotations[sidecarrewriteAppHTTPProbers] = "true"
}

// Validate returns nil due to no need for validation
func (r *TargetConfiguration) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
