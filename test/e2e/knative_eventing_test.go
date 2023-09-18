package e2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	"github.com/openshift-knative/serverless-operator/test/upgrade"
	"knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	eventingNamespace  = test.EventingNamespace
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

	ctx := context.WithValue(context.Background(), client.Key{}, caCtx.Clients.Kube)
	ctx = context.WithValue(ctx, dynamicclient.Key{}, caCtx.Clients.Dynamic)
	ctx = logging.WithLogger(ctx, logtesting.TestLogger(t))

	t.Run("verify health metrics work correctly", func(t *testing.T) {
		// Eventing should be up
		if err := monitoringe2e.VerifyHealthStatusMetric(ctx, "eventing_status", "1"); err != nil {
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
			if err := test.WithWorkloadReady(caCtx, deployment, eventingNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment, err)
			}
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		VerifyNoDisallowedImageReference(t, caCtx, eventingNamespace)
	})

	t.Run("Verify job succeeded", func(t *testing.T) {
		upgrade.VerifyPostInstallJobs(caCtx, upgrade.VerifyPostJobsConfig{
			Namespace:    eventingNamespace,
			FailOnNoJobs: true,
		})
	})

	VerifyDashboards(t, caCtx, EventingDashboards)
	VerifyNamespaceMetadata(t, caCtx, eventingNamespace)
}
