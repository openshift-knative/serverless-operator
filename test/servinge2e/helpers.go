package servinge2e

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/spoof"
)

func WaitForRouteServingText(t *testing.T, caCtx *test.Context, routeURL *url.URL, expectedText string) {
	t.Helper()
	if _, err := pkgTest.WaitForEndpointState(
		context.Background(),
		caCtx.Clients.Kube,
		t.Logf,
		routeURL,
		pkgTest.EventuallyMatchesBody(expectedText),
		"WaitForRouteToServeText",
		true,
		insecureSkipVerify(),
	); err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected text \"%s\": %v", routeURL, expectedText, err)
	}
}

func insecureSkipVerify() spoof.TransportOption {
	return func(transport *http.Transport) *http.Transport {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return transport
	}
}
