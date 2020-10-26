package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
)

const (
	eventingName      = "knative-eventing"
	eventingNamespace = "knative-eventing"
)

var knativeControlPlaneDeploymentNames = []string{
	"eventing-controller",
	"eventing-webhook",
	"imc-controller",
	"imc-dispatcher",
	"mt-broker-controller",
	"mt-broker-filter",
	"mt-broker-ingress",
	"sugar-controller",
}

func TestKnativeEventing(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)

	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	t.Run("create subscription and wait for CSV to succeed", func(t *testing.T) {
		if _, err := test.WithOperatorReady(caCtx, "serverless-operator-subscription"); err != nil {
			t.Fatal("Failed", err)
		}
	})

	t.Run("deploy knativeeventing cr and wait for it to be ready", func(t *testing.T) {
		if _, err := v1a1test.WithKnativeEventingReady(caCtx, eventingName, eventingNamespace); err != nil {
			t.Fatal("Failed to deploy KnativeEventing", err)
		}
	})

	t.Run("verify correct deployment shape", func(t *testing.T) {
		for i := range knativeControlPlaneDeploymentNames {
			deploymentName := knativeControlPlaneDeploymentNames[i]
			if _, err := test.WithDeploymentReady(caCtx, deploymentName, eventingNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		VerifyNoDisallowedImageReference(t, caCtx, eventingNamespace)
	})

	t.Run("remove knativeeventing cr", func(t *testing.T) {
		if err := v1a1test.DeleteKnativeEventing(caCtx, eventingName, eventingNamespace); err != nil {
			t.Fatal("Failed to remove Knative Eventing", err)
		}

		for i := range knativeControlPlaneDeploymentNames {
			deploymentName := knativeControlPlaneDeploymentNames[i]
			if err := test.WithDeploymentGone(caCtx, deploymentName, eventingNamespace); err != nil {
				t.Fatalf("Deployment %s is not gone: %v", deploymentName, err)
			}
		}
	})

	t.Run("undeploy serverless operator and check dependent operators removed", func(t *testing.T) {
		caCtx.Cleanup(t)
		if err := test.WaitForOperatorDepsDeleted(caCtx); err != nil {
			t.Fatalf("Operators still running: %v", err)
		}
	})
}
