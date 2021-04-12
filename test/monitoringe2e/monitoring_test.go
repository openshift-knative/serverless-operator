package monitoringe2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
)

func TestKnativeControlPlaneMetrics(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, caCtx)
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()
	t.Run("verify Serving control plane metrics work correctly", func(t *testing.T) {
		// Serving control plane metrics should work
		if err := VerifyControlPlaneMetrics(caCtx, servingMetricQueries); err != nil {
			t.Fatal("Failed to verify that Serving control plane metrics work correctly", err)
		}
	})

	t.Run("verify Eventing control plane metrics work correctly", func(t *testing.T) {
		// Eventing control plane metrics should work
		if err := VerifyControlPlaneMetrics(caCtx, eventingMetricQueries); err != nil {
			t.Fatal("Failed to verify that Eventing control plane metrics work correctly", err)
		}
	})
}
