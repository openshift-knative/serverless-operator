package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
)

const (
	servingNamespace = "knative-serving"
	haReplicas       = 2
)

func TestKnativeServing(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	t.Run("verify health metrics work correctly", func(t *testing.T) {
		// Serving should be up
		if err := monitoringe2e.VerifyHealthStatusMetric(caCtx, "serving_status", "1"); err != nil {
			t.Fatal("Failed to verify that health metrics work correctly for Serving", err)
		}
	})

	t.Run("verify correct deployment shape", func(t *testing.T) {
		// Check the desired scale of deployments in the knative serving namespace
		for _, deployment := range []string{"activator", "controller", "autoscaler-hpa"} {
			if err := test.CheckDeploymentScale(caCtx, servingNamespace, deployment, haReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings for %q: %v", deployment, err)
			}
		}
		// Check the status of deployments in the knative serving namespace
		for _, deployment := range []string{"activator", "autoscaler", "autoscaler-hpa", "controller", "webhook"} {
			if _, err := test.WithDeploymentReady(caCtx, deployment, servingNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment, err)
			}
		}
		// Check the desired scale of deployments in the ingress namespace.
		ingressDeployments := []string{"net-kourier-controller", "3scale-kourier-gateway"}
		for _, deployment := range ingressDeployments {
			if err := test.CheckDeploymentScale(caCtx, servingNamespace+"-ingress", deployment, haReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings: %v", err)
			}
		}
		// Check the status of deployments in the ingress namespace.
		for _, deployment := range ingressDeployments {
			if _, err := test.WithDeploymentReady(caCtx, deployment, servingNamespace+"-ingress"); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment, err)
			}
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		VerifyNoDisallowedImageReference(t, caCtx, servingNamespace)
	})
}
