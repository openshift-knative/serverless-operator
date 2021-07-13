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

func TestKnConsoleCLIDownload(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)

	// Verify kn ConsoleCLIDownload CO and if download links are cluster local
	ccd, err := caCtx.Clients.ConsoleCLIDownload.Get(context.Background(), "kn", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to GET kn ConsoleCLIDownload: %v", err)
	}
	// Verify the links in kn CCD CO
	if len(ccd.Spec.Links) != 5 {
		t.Fatalf("expecting 5 links for artifacts for kn ConsoleCLIDownload, found %d", len(ccd.Spec.Links))
	}
	// Verify if individual link starts with correct route
	for _, link := range ccd.Spec.Links {
		if !strings.HasPrefix(link.Href, "https://") {
			t.Fatalf("incorrect href found for kn CCD, expecting prefix %q, found link %q", "https://", link.Href)
		}
		client := &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // OCP clusters have self-signed certs by default.
			},
		}}
		h, err := client.Head(link.Href)
		if err != nil {
			t.Fatalf("Failed to perform a HEAD request for URL %q, error: %v", link.Href, err)
		}
		if h.ContentLength < 1024*1024*10 {
			t.Errorf("Failed to verify kn CCD, kn artifact %q size %d less than 10MB", link.Href, h.ContentLength)
		}
	}
}
