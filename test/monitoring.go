package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	prom "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection/clients/dynamicclient"
)

type authRoundtripper struct {
	authorization string
	inner         http.RoundTripper
}

func (a *authRoundtripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", a.authorization)
	return a.inner.RoundTrip(r)
}

func NewPrometheusClient(ctx context.Context) (promv1.API, error) {
	host, err := getPrometheusHost(ctx)
	if err != nil {
		return nil, err
	}
	bToken, err := getBearerTokenForPrometheusAccount(ctx)
	if err != nil {
		return nil, err
	}

	rt := prom.DefaultRoundTripper.(*http.Transport).Clone()
	rt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client, err := prom.NewClient(prom.Config{
		Address: "https://" + host,
		RoundTripper: &authRoundtripper{
			authorization: fmt.Sprintf("Bearer %s", bToken),
			inner:         rt,
		},
	})
	if err != nil {
		return nil, err
	}

	return promv1.NewAPI(client), nil
}

func getPrometheusHost(ctx context.Context) (string, error) {
	routeGVR := schema.GroupVersionResource{Group: "route.openshift.io", Version: "v1", Resource: "routes"}
	route, err := dynamicclient.Get(ctx).Resource(routeGVR).Namespace("openshift-monitoring").
		Get(ctx, "prometheus-k8s", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("unable to get route: %w", err)
	}
	host, _, _ := unstructured.NestedString(route.Object, "spec", "host")
	return host, nil
}

func getBearerTokenForPrometheusAccount(ctx context.Context) (string, error) {
	token, err := kubeclient.Get(ctx).CoreV1().ServiceAccounts("openshift-monitoring").
		CreateToken(context.Background(), "prometheus-k8s", &authv1.TokenRequest{}, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create prometheus token: %w", err)
	}
	return token.Status.Token, nil
}
