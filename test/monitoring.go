package test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"

	prom "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
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
	secrets, err := kubeclient.Get(ctx).CoreV1().Secrets("openshift-monitoring").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error listing secrets in namespace openshift-monitoring: %w", err)
	}
	tokenSecret := getSecretNameForToken(secrets.Items)
	if tokenSecret == "" {
		return "", errors.New("token name for prometheus-k8s service account not found")
	}
	sec, err := kubeclient.Get(ctx).CoreV1().Secrets("openshift-monitoring").Get(context.Background(), tokenSecret, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting secret %s: %w", tokenSecret, err)
	}
	tokenContents := sec.Data["token"]
	if len(tokenContents) == 0 {
		return "", fmt.Errorf("token data is missing for token %s", tokenSecret)
	}
	return string(tokenContents), nil
}

func getSecretNameForToken(secrets []corev1.Secret) string {
	for _, sec := range secrets {
		if strings.HasPrefix(sec.Name, "prometheus-k8s-token") {
			return sec.Name
		}
	}
	return ""
}
