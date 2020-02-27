package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
)

const (
	knativeEventing = "knative-eventing"
)

var knativeControlPlaneDeploymentNames = []string{
	"eventing-controller",
	"eventing-webhook",
	"imc-controller",
	"imc-dispatcher",
	"sources-controller",
}

func TestKnativeEventing(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	paCtx := test.SetupProjectAdmin(t)
	editCtx := test.SetupEdit(t)
	viewCtx := test.SetupView(t)

	defer test.CleanupAll(caCtx, paCtx, editCtx, viewCtx)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(caCtx, paCtx, editCtx, viewCtx) })

	t.Run("create subscription and wait for CSV to succeed", func(t *testing.T) {
		_, err := test.WithOperatorReady(caCtx, "serverless-operator-subscription")
		if err != nil {
			t.Fatal("Failed", err)
		}
	})

	t.Run("deploy knativeeventing cr and wait for it to be ready", func(t *testing.T) {
		_, err := v1a1test.WithKnativeEventingReady(caCtx, knativeEventing, knativeEventing)
		if err != nil {
			t.Fatal("Failed to deploy KnativeEventing", err)
		}
	})

	t.Run("verify correct deployment shape", func(t *testing.T) {
		for i := range knativeControlPlaneDeploymentNames {
			deploymentName := knativeControlPlaneDeploymentNames[i]
			_, err := test.WithDeploymentReady(caCtx, deploymentName, knativeEventing)
			if err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}

		err := test.WithDeploymentCount(caCtx, knativeEventing, len(knativeControlPlaneDeploymentNames))
		if err != nil {
			t.Fatalf("Deployment count in namespace %s is not the same as expected %d: %v", knativeEventing, len(knativeControlPlaneDeploymentNames), err)
		}
	})

	t.Run("remove knativeeventing cr", func(t *testing.T) {
		if err := v1a1test.DeleteKnativeEventing(caCtx, knativeEventing, knativeEventing); err != nil {
			t.Fatal("Failed to remove Knative Eventing", err)
		}

		for i := range knativeControlPlaneDeploymentNames {
			deploymentName := knativeControlPlaneDeploymentNames[i]
			err := test.WithDeploymentGone(caCtx, deploymentName, knativeEventing)
			if err != nil {
				t.Fatalf("Deployment %s is not gone: %v", deploymentName, err)
			}
		}

		err := test.WithDeploymentCount(caCtx, knativeEventing, 0)
		if err != nil {
			t.Fatalf("Some deployments were to be deleted but not in namespace %s. Err: %v", knativeEventing, err)
		}
	})
}
