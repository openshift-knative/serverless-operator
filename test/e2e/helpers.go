package e2e

import (
	"context"
	"fmt"
	"net/url"

	"github.com/openshift-knative/serverless-operator/test"
	v1 "github.com/openshift/api/route/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	pkgTest "knative.dev/pkg/test"
)

func setupMetricsRoute(caCtx *test.Context, name string) (*v1.Route, error) {
	metricsRoute := &v1.Route{
		ObjectMeta: meta.ObjectMeta{
			Name:      "metrics-" + name,
			Namespace: "openshift-serverless",
		},
		Spec: v1.RouteSpec{
			Port: &v1.RoutePort{
				TargetPort: intstr.FromString("8383"),
			},
			Path: "/metrics",
			To: v1.RouteTargetReference{
				Kind: "Service",
				Name: "knative-openshift-metrics",
			},
		},
	}
	r, err := caCtx.Clients.Route.Routes("openshift-serverless").Create(context.Background(), metricsRoute, meta.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating service monitor route: %v", err)
	}
	caCtx.AddToCleanup(func() error {
		caCtx.T.Logf("Cleaning up service metrics route")
		return caCtx.Clients.Route.Routes("openshift-serverless").Delete(context.Background(), "metrics", meta.DeleteOptions{})
	})
	return r, nil
}

func verifyHealthStatusMetric(caCtx *test.Context, metricsPath string, metricLabel string, expectedValue int) {
	// Check if Operator's service monitor service is available
	_, err := caCtx.Clients.Kube.CoreV1().Services("openshift-serverless").Get(context.Background(), "knative-openshift-metrics", meta.GetOptions{})
	if err != nil {
		caCtx.T.Fatalf("Error getting service monitor service: %v", err)
	}
	metricsURL, err := url.Parse(metricsPath)
	if err != nil {
		caCtx.T.Fatalf("Error parsing url for metrics: %v", err)
	}
	expectedStr := fmt.Sprintf(`knative_up{type="%s"} %d`, metricLabel, expectedValue)
	// Wait until the endpoint is actually working and we get the expected value back
	_, err = pkgTest.WaitForEndpointState(
		context.Background(),
		caCtx.Clients.Kube,
		caCtx.T.Logf,
		metricsURL,
		pkgTest.EventuallyMatchesBody(expectedStr),
		"WaitForMetricsToServeText",
		true)
	if err != nil {
		caCtx.T.Fatalf("Failed to access the operator metrics endpoint and get the metric value expected: %v", err)
	}
}
