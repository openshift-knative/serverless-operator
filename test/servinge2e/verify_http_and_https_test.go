package servinge2e

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
)

func TestKnativeServiceHTTPS(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	ksvc, err := test.WithServiceReady(caCtx, "https-service", testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Implicitly checks that HTTP works.
	WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), helloworldText)

	// Now check that HTTPS works.
	httpsURL := ksvc.Status.URL.DeepCopy()
	httpsURL.Scheme = "https"

	// First, download the cert from the host so we can trust it later.
	conn, err := tls.Dial("tcp", httpsURL.Host+":443", &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatal("Failed to connect to download certificate", err)
	}
	defer conn.Close()

	// Add the cert to our cert pool, so it's trusted.
	certPool, err := x509.SystemCertPool()
	if err != nil {
		t.Fatal("Failed to load system cert pool", err)
	}
	for _, cert := range conn.ConnectionState().PeerCertificates {
		certPool.AddCert(cert)
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certPool,
		},
	}}
	t.Log("Requesting", httpsURL.String())
	resp, err := client.Get(httpsURL.String())
	if err != nil {
		t.Fatalf("Request to %v failed, err: %v", httpsURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error("Failed to read body", err)
		}
		t.Fatalf("Response failed, status %v, body %v", resp.StatusCode, string(body))
	}
}
