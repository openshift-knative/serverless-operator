package kourier

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
	pkgTest "knative.dev/pkg/test"
)

func TestKnativeServiceHTTPS(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	ksvc, err := test.WithServiceReady(caCtx, "https-service", testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Implicitly checks that HTTPS works.
	servinge2e.WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), helloworldText)

	// Now check that HTTP works.
	httpURL := ksvc.Status.URL.DeepCopy()

	httpURL.Scheme = "http"
	if _, err := pkgTest.WaitForEndpointState(
		context.Background(),
		caCtx.Clients.Kube,
		t.Logf,
		httpURL.URL(),
		pkgTest.EventuallyMatchesBody(helloworldText),
		"WaitForRouteToServeText",
		true,
	); err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected text %q: %v", httpURL, helloworldText, err)
	}

}
