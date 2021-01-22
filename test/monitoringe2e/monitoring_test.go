package monitoringe2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
)

func TestKnativeServingControlPlaneMetrics(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, caCtx)
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()
	t.Run("verify Serving control plane metrics work correctly", func(t *testing.T) {
		// Serving control plane metrics should work
		if err := VerifyServingControlPlaneMetrics(caCtx); err != nil {
			t.Fatal("Failed to verify that control plane metrics work correctly", err)
		}
	})
}
