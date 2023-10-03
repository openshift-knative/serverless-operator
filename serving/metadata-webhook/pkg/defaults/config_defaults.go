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
func (r *TargetConfiguration) SetDefaults(_ context.Context) {
	if r.Spec.Template.Annotations == nil {
		r.Spec.Template.Annotations = make(map[string]string)
	}

	r.Spec.Template.Annotations[sidecarInject] = "true"
	r.Spec.Template.Annotations[sidecarrewriteAppHTTPProbers] = "true"
}

// Validate returns nil due to no need for validation
func (r *TargetConfiguration) Validate(_ context.Context) *apis.FieldError {
	return nil
}
