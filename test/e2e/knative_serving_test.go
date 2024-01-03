package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	"github.com/openshift-knative/serverless-operator/test/v1beta1"
	"knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/ptr"
)

const (
	servingNamespace  = test.ServingNamespace
	servingHAReplicas = 2
)

func TestKnativeServing(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	ctx := context.WithValue(context.Background(), client.Key{}, caCtx.Clients.Kube)
	ctx = context.WithValue(ctx, dynamicclient.Key{}, caCtx.Clients.Dynamic)
	ctx = logging.WithLogger(ctx, logtesting.TestLogger(t))

	t.Run("verify health metrics work correctly", func(t *testing.T) {
		// Serving should be up
		if err := monitoringe2e.VerifyHealthStatusMetric(ctx, "serving_status", "1"); err != nil {
			t.Fatal("Failed to verify that health metrics work correctly for Serving", err)
		}
	})

	t.Run("verify correct deployment shape", func(t *testing.T) {
		servingDeployments := []test.Deployment{
			{Name: "activator"},
			{Name: "autoscaler"},
			{Name: "autoscaler-hpa"},
			{Name: "controller"},
			{Name: "webhook"},
		}
		if err := v1beta1.UpdateServingExpectedScale(caCtx,
			servingNamespace, servingNamespace, servingDeployments, ptr.Int32(servingHAReplicas)); err != nil {
			t.Fatalf("Failed to update deployment scale: %v", err)
		}

		for _, deployment := range servingDeployments {
			// Check the desired scale of deployments in the knative serving namespace
			if err := test.CheckDeploymentScale(caCtx, servingNamespace, deployment.Name, *deployment.ExpectedScale); err != nil {
				t.Fatalf("Failed to verify default HA settings for %q: %v", deployment.Name, err)
			}

			// Check the status of deployments in the knative serving namespace
			if err := test.WithWorkloadReady(caCtx, deployment.Name, servingNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment.Name, err)
			}
		}

		ingressDeployments := []test.Deployment{
			{Name: "net-kourier-controller"},
			{Name: "3scale-kourier-gateway"},
		}
		ingressNamespace := servingNamespace + "-ingress"

		// If FULL_MESH is true, net-istio is used instead of net-kourier.
		if os.Getenv("FULL_MESH") == "true" {
			ingressDeployments = []test.Deployment{
				{Name: "net-istio-controller"},
				{Name: "net-istio-webhook"},
			}
			ingressNamespace = servingNamespace
		}

		if err := v1beta1.UpdateServingExpectedScale(caCtx,
			servingNamespace, servingNamespace, ingressDeployments, ptr.Int32(servingHAReplicas)); err != nil {
			t.Fatalf("Failed to update deployment scale: %v", err)
		}

		// Check the desired scale of deployments in the ingress namespace.
		for _, deployment := range ingressDeployments {
			if err := test.CheckDeploymentScale(caCtx, ingressNamespace, deployment.Name, *deployment.ExpectedScale); err != nil {
				t.Fatalf("Failed to verify default HA settings for %q: %v", deployment.Name, err)
			}
		}
		// Check the status of deployments in the ingress namespace.
		for _, deployment := range ingressDeployments {
			if err := test.WithWorkloadReady(caCtx, deployment.Name, ingressNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment.Name, err)
			}
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		VerifyNoDisallowedImageReference(t, caCtx, servingNamespace)
	})
}
