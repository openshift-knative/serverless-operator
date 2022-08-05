package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	"github.com/openshift-knative/serverless-operator/test/upgrade"
)

const (
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

	t.Run("verify health metrics work correctly", func(t *testing.T) {
		// Eventing should be up
		if err := monitoringe2e.VerifyHealthStatusMetric(caCtx, "eventing_status", "1"); err != nil {
			t.Fatal("Failed to verify that health metrics work correctly for Eventing", err)
		}
	})

	t.Run("verify correct deployment shape", func(t *testing.T) {
		// Check the desired scale of deployments in the knative eventing namespace
		for _, deployment := range []string{"eventing-controller", "eventing-webhook", "imc-controller", "imc-dispatcher", "mt-broker-controller"} {
			if err := test.CheckDeploymentScale(caCtx, eventingNamespace, deployment, eventingHaReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings for %q: %v", deployment, err)
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

	t.Run("Verify sugar controller deletion", func(t *testing.T) {
		if err := test.CheckNoDeployment(caCtx.Clients.Kube, eventingNamespace, "sugar-controller"); err != nil {
			t.Errorf("sugar-controller is still present: %v", err)
		}
	})

	t.Run("Verify job succeeded", func(t *testing.T) {
		upgrade.VerifyPostInstallJobs(caCtx, upgrade.VerifyPostJobsConfig{
			Namespace:    eventingNamespace,
			FailOnNoJobs: true,
		})
	})
}
