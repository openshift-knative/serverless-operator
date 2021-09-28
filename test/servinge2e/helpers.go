package servinge2e

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/logging"
	"knative.dev/pkg/test/spoof"
)

const (
	testNamespace2        = "serverless-tests2"
	image                 = "gcr.io/knative-samples/helloworld-go"
	helloworldService     = "helloworld-go"
	helloworldService2    = "helloworld-go2"
	kubeHelloworldService = "kube-helloworld-go"
	helloworldText        = "Hello World!"

	caSecretNamespace = "cert-manager"
	caSecretName      = "ca-key-pair"
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
		addRootCAtoTransport(context.Background(), t.Logf, caCtx.Clients),
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

// addRootCAtoTransport returns TransportOption when HTTPS option is true. Otherwise it returns plain spoof.TransportOption.
func addRootCAtoTransport(ctx context.Context, logf logging.FormatLogger, clients *test.Clients) spoof.TransportOption {
	return func(transport *http.Transport) *http.Transport {
		transport.TLSClientConfig = TLSClientConfig(ctx, logf, clients)
		return transport
	}
}

func TLSClientConfig(ctx context.Context, logf logging.FormatLogger, clients *test.Clients) *tls.Config {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if !rootCAs.AppendCertsFromPEM(PemDataFromSecret(ctx, logf, clients, caSecretNamespace, caSecretName)) {
		logf("Failed to add the certificate to the root CA")
	}
	return &tls.Config{RootCAs: rootCAs}
}

// PemDataFromSecret gets pem data from secret.
func PemDataFromSecret(ctx context.Context, logf logging.FormatLogger, clients *test.Clients, ns, secretName string) []byte {
	secret, err := clients.Kube.CoreV1().Secrets(ns).Get(
		ctx, secretName, metav1.GetOptions{})
	if err != nil {
		logf("Failed to get Secret %s: %v", secretName, err)
		return []byte{}
	}
	return secret.Data[corev1.TLSCertKey]
}
