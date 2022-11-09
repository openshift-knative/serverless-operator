package e2e

import (
	"os"
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
		servingDeployments := []string{"activator", "autoscaler", "autoscaler-hpa", "controller", "domain-mapping", "domainmapping-webhook", "webhook"}

		for _, deployment := range servingDeployments {
			// Check the desired scale of deployments in the knative serving namespace
			if err := test.CheckDeploymentScale(caCtx, servingNamespace, deployment, haReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings for %q: %v", deployment, err)
			}

			// Check the status of deployments in the knative serving namespace
			if err := test.WithWorkloadReady(caCtx, deployment, servingNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment, err)
			}
		}

		ingressDeployments := []string{"net-kourier-controller", "3scale-kourier-gateway"}
		ingressNamespace := servingNamespace + "-ingress"

		// If FULL_MESH is true, net-istio is used instead of net-kourier.
		if os.Getenv("FULL_MESH") == "true" {
			ingressDeployments = []string{"net-istio-controller", "net-istio-webhook"}
			ingressNamespace = servingNamespace
		}

		// Check the desired scale of deployments in the ingress namespace.
		for _, deployment := range ingressDeployments {
			if err := test.CheckDeploymentScale(caCtx, ingressNamespace, deployment, haReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings: %v", err)
			}
		}
		// Check the status of deployments in the ingress namespace.
		for _, deployment := range ingressDeployments {
			if err := test.WithWorkloadReady(caCtx, deployment, ingressNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment, err)
			}
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		VerifyNoDisallowedImageReference(t, caCtx, servingNamespace)
	})
}
