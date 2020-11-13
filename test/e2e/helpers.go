package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	v1 "github.com/openshift/api/route/v1"
	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	pkgTest "knative.dev/pkg/test"
)

func extracMetrictData(body io.Reader, metricName string) (*float64, error) {
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(body)
	if err != nil {
		return nil, fmt.Errorf("reading text format failed: %v", err)
	}
	pm := prometheusMetric(metricFamilies, "knative_up")
	if pm == nil {
		return nil, errors.New("could not get metric family `knative_up` from prometheus exported metrics")
	}
	status := getMetricValueByTypeLabel(metricName, pm)
	if status == nil {
		return nil, fmt.Errorf("could not get metric type `%s` from prometheus metric `knative_up`", metricName)
	}
	return status, nil
}

func prometheusMetric(metricFamilies map[string]*ioprometheusclient.MetricFamily, key string) []*ioprometheusclient.Metric {
	if metric, ok := metricFamilies[key]; ok && len(metric.Metric) > 0 {
		return metric.Metric
	}
	return nil
}

func getMetricValueByTypeLabel(label string, metrics []*ioprometheusclient.Metric) *float64 {
	for _, metric := range metrics {
		if len(metric.Label) == 0 {
			return nil
		}
		// we expect one label
		if metric.Label[0] == nil {
			return nil
		}
		if metric.Label[0].Name == nil || metric.Label[0].Value == nil {
			return nil
		}
		if (*metric.Label[0].Name) == "type" && (*metric.Label[0].Value) == label {
			return metric.Gauge.Value
		}
	}
	return nil
}

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

func verifyHealthStatusMetric(caCtx *test.Context, metricsPath string, metricName string, expectedValue int, t *testing.T) {
	// Check if Operator's service monitor service is available
	_, err := caCtx.Clients.Kube.CoreV1().Services("openshift-serverless").Get(context.Background(), "knative-openshift-metrics", meta.GetOptions{})
	if err != nil {
		t.Errorf("error getting service monitor service: %v", err)
		return
	}

	metricsURL, err := url.Parse(metricsPath)
	if err != nil {
		t.Errorf("error parsing url for metrics: %v", err)
		return
	}

	// Wait until the endpoint is actually working
	resp, err := pkgTest.WaitForEndpointState(
		context.Background(),
		&pkgTest.KubeClient{Kube: caCtx.Clients.Kube},
		t.Logf,
		metricsURL,
		pkgTest.EventuallyMatchesBody("# TYPE knative_up gauge"),
		"WaitForMetricsToServeText",
		true)

	if err != nil {
		t.Errorf("the operator metrics endpoint is not accessible: %v", err)
	}
	stat, err := extracMetrictData(bytes.NewReader(resp.Body), metricName)
	if err != nil {
		t.Fatal("Failed to get metrics from operator's prometheus endpoint", err)
	}
	t.Errorf("Got = %v, want: %v for Eventing health status", stat, expectedValue)
}
