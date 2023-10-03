package defaults

import (
	"context"

	"knative.dev/pkg/apis"
	servingv1beta1 "knative.dev/serving/pkg/apis/serving/v1beta1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetDomainMapping is a wrapper around Configuration.
type TargetDomainMapping struct {
	servingv1beta1.DomainMapping `json:",inline"`
}

// Verify that Deployment adheres to the appropriate interfaces.
var (
	// Check that Deployment can be defaulted.
	_ apis.Defaultable = (*TargetDomainMapping)(nil)
	_ apis.Validatable = (*TargetDomainMapping)(nil)
)

// SetDefaults implements apis.Defaultable
func (r *TargetDomainMapping) SetDefaults(_ context.Context) {
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
	r.Annotations[openshiftPassthrough] = "true"
}

// Validate returns nil due to no need for validation
func (r *TargetDomainMapping) Validate(_ context.Context) *apis.FieldError {
	return nil
}
