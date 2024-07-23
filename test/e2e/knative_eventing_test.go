package e2e

import (
	"context"
	"fmt"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	"knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/ptr"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	"github.com/openshift-knative/serverless-operator/test/upgrade"
	"github.com/openshift-knative/serverless-operator/test/v1beta1"
)

const (
	eventingNamespace  = test.EventingNamespace
	eventingHaReplicas = 2
)

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
		var eventingDeployments = []test.Deployment{
			{Name: "eventing-controller"},
			{Name: "eventing-webhook"},
			{Name: "imc-controller"},
			{Name: "imc-dispatcher"},
			{Name: "mt-broker-controller"},
			{Name: "mt-broker-filter"},
			{Name: "mt-broker-ingress"},
		}
		if err := v1beta1.UpdateEventingExpectedScale(caCtx,
			eventingNamespace, eventingNamespace, eventingDeployments, ptr.Int32(eventingHaReplicas)); err != nil {
			t.Fatalf("Failed to update deployment scale: %v", err)
		}
		// Check the desired scale of deployments in the knative eventing namespace
		for _, deployment := range eventingDeployments {
			if err := test.CheckDeploymentScale(caCtx, eventingNamespace, deployment.Name, deployment.ExpectedScale); err != nil {
				t.Fatalf("Failed to verify default HA settings for %q: %v", deployment.Name, err)
			}
		}
		// Check the status of deployments in the knative eventing namespace
		for _, deployment := range eventingDeployments {
			if err := test.WithWorkloadReady(caCtx, deployment.Name, eventingNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment.Name, err)
			}
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		VerifyNoDisallowedImageReference(t, caCtx, eventingNamespace)
	})

	t.Run("Verify job succeeded", func(_ *testing.T) {
		upgrade.VerifyPostInstallJobs(caCtx, upgrade.VerifyPostJobsConfig{
			Namespace:    eventingNamespace,
			FailOnNoJobs: true,
			ValidateJob: func(j batchv1.Job) error {
				if j.Spec.TTLSecondsAfterFinished != nil {
					return fmt.Errorf("job %s/%s has TTLSecondsAfterFinished", eventingNamespace, j.Name)
				}
				return nil
			},
		})
	})

	VerifyDashboards(t, caCtx, EventingDashboards)
	VerifyNamespaceMetadata(t, caCtx, eventingNamespace)
}
