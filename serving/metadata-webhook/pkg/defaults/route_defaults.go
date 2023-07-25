package defaults

import (
	"context"

	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetRoute is a wrapper around Configuration.
type TargetRoute struct {
	servingv1.Route `json:",inline"`
}

// Verify that Deployment adheres to the appropriate interfaces.
var (
	// Check that Deployment can be defaulted.
	_ apis.Defaultable = (*TargetRoute)(nil)
	_ apis.Validatable = (*TargetRoute)(nil)
)

// SetDefaults implements apis.Defaultable
func (r *TargetRoute) SetDefaults(_ context.Context) {
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
	r.Annotations[openshiftPassthrough] = "true"
}

// Validate returns nil due to no need for validation
func (r *TargetRoute) Validate(_ context.Context) *apis.FieldError {
	return nil
}
