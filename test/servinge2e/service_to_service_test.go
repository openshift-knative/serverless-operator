package servinge2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	network "knative.dev/networking/pkg"
	nv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/apis/autoscaling"
)

// Smoke tests for networking which access public and cluster-local
// services from within the cluster.
func TestServiceToServiceCalls(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	tests := []testCase{{
		// Requests go via gateway -> activator -> pod.
		name: "service-call-via-activator",
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "-1",
		},
	}, {
		// Requests go via gateway -> pod (activator should be skipped if burst
		// capacity is disabled and there is at least 1 replica).
		name: "service-call-without-activator",
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "0",
			autoscaling.MinScaleAnnotationKey:  "1",
		},
	}, {
		name: "cluster-local-via-activator",
		labels: map[string]string{
			network.VisibilityLabelKey: string(nv1alpha1.IngressVisibilityClusterLocal),
		},
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "-1",
		},
	}, {
		name: "cluster-local-without-activator",
		labels: map[string]string{
			network.VisibilityLabelKey: string(nv1alpha1.IngressVisibilityClusterLocal),
		},
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "0",
			autoscaling.MinScaleAnnotationKey:  "1",
		},
	}}

	for _, scenario := range tests {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			testServiceToService(t, caCtx, testNamespace, scenario)
		})
	}
}
