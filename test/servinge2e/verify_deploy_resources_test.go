package servinge2e

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	knativeServing = "knative-serving"
)

func TestConsoleCLIDownloadAndDeploymentResources(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	// Check the status of Service for kn ConsoleCLIDownload
	service, err := test.WaitForServiceState(caCtx, "kn-cli", knativeServing, test.IsServiceReady)
	if err != nil {
		t.Fatalf("failed to verify kn ConcoleCLIDownload Deployment: %v", err)
	}
	// Verify that Service URL for kn ConsoleCLIDownload is present and has a host
	host := service.Status.URL.Host
	if host == "" {
		t.Fatalf("failed to verify kn ConsoleCLIDownload Service URL is present: %v", err)
	}
	// Verify kn ConsoleCLIDownload CO and if download links are cluster local
	ccd, err := caCtx.Clients.ConsoleCLIDownload.Get(context.Background(), "kn", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unable to GET kn ConsoleCLIDownload CO 'kn': %v", err)
	}
	// Verify the links in kn CCD CO
	if len(ccd.Spec.Links) != 6 {
		t.Fatalf("expecting 6 links for artifacts for kn ConsoleCLIDownload, found %d", len(ccd.Spec.Links))
	}
	// Verify if individual link starts with correct route
	protocol := "https://"
	if !strings.HasPrefix(host, protocol) {
		host = protocol + host
	}
	for _, link := range ccd.Spec.Links {
		if !strings.HasPrefix(link.Href, host) {
			t.Fatalf("incorrect href found for kn CCD, expecting prefix %s, found link %s", host, link.Href)
		}
		client := &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // OCP clusters have self-signed certs by default.
			},
		}}
		h, err := client.Head(link.Href)
		if err != nil {
			t.Fatalf("failed to HEAD request for URL %s, error: %v", link.Href, err)
		}
		if h.ContentLength < 1024*1024*10 {
			t.Fatalf("failed to verify kn CCD, kn artifact %s size less than 10MB", link.Href)
		}
	}
}
