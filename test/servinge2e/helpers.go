package servinge2e

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	v1 "github.com/openshift/api/route/v1"
	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	pkgTest "knative.dev/pkg/test"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func WaitForRouteServingText(t *testing.T, caCtx *test.Context, routeURL *url.URL, expectedText string) {
	t.Helper()
	if _, err := pkgTest.WaitForEndpointState(
		context.Background(),
		&pkgTest.KubeClient{Kube: caCtx.Clients.Kube},
		t.Logf,
		routeURL,
		pkgTest.EventuallyMatchesBody(expectedText),
		"WaitForRouteToServeText",
		true); err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected text \"%s\": %v", routeURL, expectedText, err)
	}
}

func withServiceReadyOrFail(ctx *test.Context, service *servingv1.Service) *servingv1.Service {
	service, err := ctx.Clients.Serving.ServingV1().Services(service.Namespace).Create(context.Background(), service, meta.CreateOptions{})
	if err != nil {
		ctx.T.Fatalf("Error creating ksvc: %v", err)
	}

	// Let the ksvc be deleted after test
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Knative Service '%s/%s'", service.Namespace, service.Name)
		return ctx.Clients.Serving.ServingV1().Services(service.Namespace).Delete(context.Background(), service.Name, meta.DeleteOptions{})
	})

	service, err = test.WaitForServiceState(ctx, service.Name, service.Namespace, test.IsServiceReady)
	if err != nil {
		ctx.T.Fatalf("Error waiting for ksvc readiness: %v", err)
	}

	return service
}

type telemetryStat struct {
	services       float64
	routes         float64
	configurations float64
	revisions      float64
}

func getMetricsEndpointPath() string {
	portAndPath := strconv.Itoa(8383) + "/metrics"
	metricsPath := "http://knative-openshift-metrics.openshift-serverless:" + portAndPath
	return metricsPath
}
func fetchTelemetryMetrics() (*telemetryStat, error) {
	metricsPath := getMetricsEndpointPath()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, metricsPath, nil)
	if err != nil {
		return nil, err
	}
	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	stat, err := extracMetrictData(resp.Body)
	if err != nil {
		return nil, err
	}
	return stat, nil
}

func extracMetrictData(body io.Reader) (*telemetryStat, error) {
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(body)
	if err != nil {
		return nil, fmt.Errorf("reading text format failed: %v", err)
	}
	tel := telemetryStat{}
	pm := prometheusMetric(metricFamilies, "serverless_telemetry")
	if pm == nil {
		return nil, fmt.Errorf("could not get metric family from prometheus exported metrics")
	}
	services := getMetricValueByTypeLabel("service", pm)
	if services == nil {
		return nil, errors.New("could not get metric `service` from prometheus exported metrics")
	}
	tel.services = *services
	configs := getMetricValueByTypeLabel("configuration", pm)
	if configs == nil {
		return nil, errors.New("could not get metric `configuration` from prometheus exported metrics")
	}
	tel.configurations = *configs
	revisions := getMetricValueByTypeLabel("revision", pm)
	if revisions == nil {
		return nil, errors.New("could not get metric `revision` from prometheus exported metrics")
	}
	tel.revisions = *revisions
	routes := getMetricValueByTypeLabel("route", pm)
	if routes == nil {
		return nil, errors.New("could not get metric `route` from prometheus exported metrics")
	}
	tel.routes = *routes
	return &tel, nil
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

func setupMetricsRoute(caCtx *test.Context) error {
	metricsRoute := &v1.Route{
		ObjectMeta: meta.ObjectMeta{
			Name:      "metrics",
			Namespace: "openshift-serverless",
		},
		Spec: v1.RouteSpec{
			Host: "knative-openshift-metrics",
			Port: &v1.RoutePort{
				TargetPort: intstr.FromString("8383"),
			},
			Path: "/metrics",
			To: v1.RouteTargetReference{
				Kind: "Service",
				Name: "knative-openshift-metrics",
			},
			TLS: &v1.TLSConfig{
				Termination:                   v1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: v1.InsecureEdgeTerminationPolicyAllow,
			},
			WildcardPolicy: v1.WildcardPolicyNone,
		},
	}

	_, err := caCtx.Clients.Route.Routes("openshift-serverless").Create(context.Background(), metricsRoute, meta.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating service monitor route: %v", err)
	}
	caCtx.AddToCleanup(func() error {
		caCtx.T.Logf("Cleaning up service metrics route")
		return caCtx.Clients.Route.Routes("openshift-serverless").Delete(context.Background(), "metrics", meta.DeleteOptions{})
	})
	return nil
}
