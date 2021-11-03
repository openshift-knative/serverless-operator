package servinge2e

import (
	"context"
	"net/url"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/spoof"
	servingTest "knative.dev/serving/test"
)

const (
	testNamespace2        = "serverless-tests2"
	image                 = "gcr.io/knative-samples/helloworld-go"
	helloworldService     = "helloworld-go"
	helloworldService2    = "helloworld-go2"
	kubeHelloworldService = "kube-helloworld-go"
	helloworldText        = "Hello World!"
)

func WaitForRouteServingText(t *testing.T, caCtx *test.Context, routeURL *url.URL, expectedText string) {
	t.Helper()
	if _, err := pkgTest.CheckEndpointState(
		context.Background(),
		caCtx.Clients.Kube,
		t.Logf,
		routeURL,
		spoof.MatchesBody(expectedText),
		"WaitForRouteToServeText",
		true,
		servingTest.AddRootCAtoTransport(context.Background(), t.Logf, &servingTest.Clients{KubeClient: caCtx.Clients.Kube}, true),
	); err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected text %q: %v", routeURL, expectedText, err)
	}
}
