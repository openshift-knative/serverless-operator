package kourier

import (
	"net/http"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
	"knative.dev/networking/pkg/apis/networking"
	pkgTest "knative.dev/pkg/test"
)

func TestKnativeServiceHTTPRedirect(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	ksvc := test.Service("redirect-service", test.Namespace, pkgTest.ImagePath(test.HelloworldGoImg), nil, nil)
	ksvc.ObjectMeta.Annotations = map[string]string{networking.HTTPProtocolAnnotationKey: "redirected"}
	ksvc = test.WithServiceReadyOrFail(caCtx, ksvc)

	// Implicitly checks that HTTPS works.
	servinge2e.WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), servinge2e.HelloworldText)

	// Now check HTTP request.
	httpURL := ksvc.Status.URL.DeepCopy()
	httpURL.Scheme = "http"

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// Do not follow redirect.
			return http.ErrUseLastResponse
		},
	}

	t.Log("Requesting", httpURL.String())
	resp, err := client.Get(httpURL.String())
	if err != nil {
		t.Fatalf("Request to %v failed, err: %v", httpURL, err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("The Route at domain %s didn't serve the expected status code got=%v, want=%v", httpURL, resp.StatusCode, http.StatusFound)
	}
}
