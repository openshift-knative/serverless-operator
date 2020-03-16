package e2e

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"

	"github.com/openshift/console-operator/test/e2e/framework"
)

// a request with a route & the bearer token
// this only works if test are run as a token user (example: kube:admin)
func getRequestWithToken(t *testing.T, url, bearer string) *http.Request {
	req := getRequest(t, url)
	req.Header.Add("Authorization", bearer)
	return req
}

func getRequest(t *testing.T, url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("%s\n", err)
	}
	return req
}

func getInsecureClient() *http.Client {
	// ignore self signed certs for testing purposes
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// gets us a token from the kubeconfig file
func getBearerToken(t *testing.T) string {
	tokenProvider, err := framework.GetConfig()
	if err != nil {
		t.Fatalf("error, can't get config: %s", err)
	}

	// use this for
	// tokenProvider.TLSClientConfig

	// build the request header with a token from the kubeconfig
	return fmt.Sprintf("Bearer %s", tokenProvider.BearerToken)
}

func httpOK(resp *http.Response) bool {
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return true
	}
	return false
}
