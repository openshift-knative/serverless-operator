package kourier

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
)

func TestKnativeServiceHTTPS(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	ksvc, err := test.WithServiceReady(caCtx, "https-service", testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Checks that HTTPS works.
	servinge2e.WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), helloworldText)

	// Now check that HTTP works.
	httpsURL := ksvc.Status.URL.DeepCopy()

	httpsURL.Scheme = "http"
	// Implicitly checks that HTTP works.
	servinge2e.WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), helloworldText)
}
