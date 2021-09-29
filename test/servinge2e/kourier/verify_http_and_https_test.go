package kourier

import (
	"context"
	"net/http"
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

func TestKnativeServiceHTTPRedirect(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	ksvc := test.Service("redirect-service", testNamespace, image, map[string]string{"networking.knative.dev/httpOption": "redirected"})
	ksvc = withServiceReadyOrFail(caCtx, ksvc)

	// Implicitly checks that HTTPS works.
	servinge2e.WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), helloworldText)

	// Now check HTTP request.
	httpURL := ksvc.Status.URL.DeepCopy()
	httpURL.Scheme = "http"

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Do not follow redirect.
			return http.ErrUseLastResponse
		},
	}

	t.Log("Requesting", httpURL.String())
	resp, err := client.Get(httpURL.String())
	if err != nil {
		t.Fatalf("Request to %v failed, err: %v", httpURL, err)
	}
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("The Route at domain %s didn't serve the expected status code got=%v, want=%v", httpURL, resp.StatusCode, http.StatusMovedPermanently)
	}
}
