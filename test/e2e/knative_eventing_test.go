package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
)

const (
	eventingName       = "knative-eventing"
	eventingNamespace  = "knative-eventing"
	eventingHaReplicas = 2
)

var knativeEventingControlPlaneDeploymentNames = []string{
	"eventing-controller",
	"eventing-webhook",
	"imc-controller",
	"imc-dispatcher",
	"mt-broker-controller",
	"mt-broker-filter",
	"mt-broker-ingress",
}

func TestKnativeEventing(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	t.Run("create subscription and wait for CSV to succeed", func(t *testing.T) {
		if _, err := test.WithOperatorReady(caCtx, test.Flags.Subscription); err != nil {
			t.Fatal("Failed", err)
		}
	})

	t.Run("deploy knativeeventing cr and wait for it to be ready", func(t *testing.T) {
		if _, err := v1a1test.WithKnativeEventingReady(caCtx, eventingName, eventingNamespace); err != nil {
			t.Fatal("Failed to deploy KnativeEventing", err)
		}
	})

	t.Run("verify health metrics work correctly", func(t *testing.T) {
		// Eventing should be up
		if err := monitoringe2e.VerifyHealthStatusMetric(caCtx, "eventing_status", "1"); err != nil {
			t.Fatal("Failed to verify that health metrics work correctly for Eventing", err)
		}
	})

	t.Run("verify correct deployment shape", func(t *testing.T) {
		// Check the status of scaled deployments in the knative eventing namespace
		for _, deployment := range []string{"eventing-controller", "eventing-webhook", "imc-controller", "imc-dispatcher", "mt-broker-controller"} {
			if err := test.CheckDeploymentScale(caCtx, eventingNamespace, deployment, eventingHaReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings for `%s`: %v", deployment, err)
			}
		}
		// Check the status of deployments in the knative eventing namespace
		for _, deployment := range knativeEventingControlPlaneDeploymentNames {
			if _, err := test.WithDeploymentReady(caCtx, deployment, eventingNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment, err)
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

		for i := range knativeEventingControlPlaneDeploymentNames {
			deploymentName := knativeEventingControlPlaneDeploymentNames[i]
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
