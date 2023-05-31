package monitoringe2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestKnativeMetrics(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, caCtx)
	}
	test.CleanupOnInterrupt(t, cleanup)

	ctx := context.Background()
	ctx = context.WithValue(ctx, client.Key{}, caCtx.Clients.Kube)
	ctx = context.WithValue(ctx, dynamicclient.Key{}, caCtx.Clients.Dynamic)
	ctx = logging.WithLogger(ctx, logtesting.TestLogger(t))

	defer cleanup()
	t.Run("verify Serving control plane metrics work correctly", func(t *testing.T) {
		// Serving control plane metrics should work
		if err := VerifyMetrics(ctx, servingMetricQueries); err != nil {
			t.Fatal("Failed to verify that Serving control plane metrics work correctly", err)
		}
	})

	t.Run("verify Eventing control plane metrics work correctly", func(t *testing.T) {
		// Eventing control plane metrics should work
		if err := VerifyMetrics(ctx, eventingMetricQueries); err != nil {
			t.Fatal("Failed to verify that Eventing control plane metrics work correctly", err)
		}
	})

	t.Run("verify Knative operators and Openshift ingress metrics work correctly", func(t *testing.T) {
		// Knative operators and Openshift ingress metrics should work
		if err := VerifyMetrics(ctx, serverlessComponentQueries); err != nil {
			t.Fatal("Failed to verify that Knative operators and Openshift ingress metrics work correctly", err)
		}
	})
}
