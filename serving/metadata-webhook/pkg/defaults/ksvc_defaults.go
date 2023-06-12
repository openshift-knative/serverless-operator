package defaults

import (
	"context"

	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	openshiftPassthrough = "serving.knative.openshift.io/enablePassthrough"

	sidecarInject                = "sidecar.istio.io/inject"
	sidecarrewriteAppHTTPProbers = "sidecar.istio.io/rewriteAppHTTPProbers"
	proxyIstioConfig             = "proxy.istio.io/config"

	maistraProxyEnv          = "sidecar.maistra.io/proxyEnv"
	terminationDrainDuration = `{ "TERMINATION_DRAIN_DURATION_SECONDS": "20" }`
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetKService is a wrapper around KService.
type TargetKService struct {
	servingv1.Service `json:",inline"`
}

// Verify that Deployment adheres to the appropriate interfaces.
var (
	// Check that Deployment can be defaulted.
	_ apis.Defaultable = (*TargetKService)(nil)
	_ apis.Validatable = (*TargetKService)(nil)
)

// SetDefaults implements apis.Defaultable
func (r *TargetKService) SetDefaults(ctx context.Context) {
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
	r.Annotations[openshiftPassthrough] = "true"

	if r.Spec.Template.Annotations == nil {
		r.Spec.Template.Annotations = make(map[string]string)
	}

	r.Spec.Template.Annotations[sidecarInject] = "true"
	r.Spec.Template.Annotations[sidecarrewriteAppHTTPProbers] = "true"
	r.Spec.Template.Annotations[maistraProxyEnv] = terminationDrainDuration
	r.Spec.Template.Annotations[proxyIstioConfig] = `{ "holdApplicationUntilProxyStarts": true }`
}

// Validate returns nil due to no need for validation
func (r *TargetKService) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
