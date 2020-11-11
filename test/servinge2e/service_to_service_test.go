package servinge2e

import (
	"context"
	"net/url"
	"testing"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgTest "knative.dev/pkg/test"

	"github.com/openshift-knative/serverless-operator/test"
	network "knative.dev/networking/pkg"
	nv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/apis/autoscaling"
)

// Smoke tests for networking which access public and cluster-local
// services from within the cluster.
func TestServiceToServiceCalls(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	err := setupMetricsRoute(caCtx)
	if err != nil {
		t.Errorf("error creating metrics service route: %v", err)
		return
	}
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	tests := []testCase{{
		// Requests go via gateway -> activator -> pod.
		name: "service-call-via-activator",
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "-1",
		},
		expectedMetricCount: 1,
	}, {
		// Requests go via gateway -> pod (activator should be skipped if burst
		// capacity is disabled and there is at least 1 replica).
		name: "service-call-without-activator",
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "0",
			autoscaling.MinScaleAnnotationKey:  "1",
		},
		expectedMetricCount: 2,
	}, {
		name: "cluster-local-via-activator",
		labels: map[string]string{
			network.VisibilityLabelKey: string(nv1alpha1.IngressVisibilityClusterLocal),
		},
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "-1",
		},
		expectedMetricCount: 3,
	}, {
		name: "cluster-local-without-activator",
		labels: map[string]string{
			network.VisibilityLabelKey: string(nv1alpha1.IngressVisibilityClusterLocal),
		},
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "0",
			autoscaling.MinScaleAnnotationKey:  "1",
		},
		expectedMetricCount: 4,
	}}

	for _, scenario := range tests {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			testServiceToService(t, caCtx, testNamespace, scenario)
		})
	}
}

func testServiceToService(t *testing.T, ctx *test.Context, namespace string, tc testCase) {
	// Create a ksvc with the specified annotations and labels
	service := test.Service(tc.name, namespace, helloworldImage, tc.annotations)
	service.ObjectMeta.Labels = tc.labels

	service = withServiceReadyOrFail(ctx, service)
	serviceURL := service.Status.URL.URL()

	// For cluster-local ksvc, we deploy an "HTTP proxy" service, and request that one instead
	if service.GetLabels()[network.VisibilityLabelKey] == string(nv1alpha1.IngressVisibilityClusterLocal) {
		// Deploy an "HTTP proxy" towards the ksvc (using an httpproxy image from knative-serving testsuite)
		httpProxy := withServiceReadyOrFail(ctx, httpProxyService(tc.name+"-proxy", namespace, service.Status.URL.Host))
		serviceURL = httpProxy.Status.URL.URL()
	}

	// Verify the service is actually accessible from the outside
	if _, err := pkgTest.WaitForEndpointState(
		context.Background(),
		&pkgTest.KubeClient{Kube: ctx.Clients.Kube},
		t.Logf,
		serviceURL,
		pkgTest.EventuallyMatchesBody(helloworldText),
		"WaitForRouteToServeText",
		true); err != nil {
		t.Errorf("the Route at domain %s didn't serve the expected text %q: %v", service.Status.URL.URL(), helloworldText, err)
	}

	// Check if service monitor service is available, at this point it should be present
	_, err := ctx.Clients.Kube.CoreV1().Services("openshift-serverless").Get(context.Background(), "knative-openshift-metrics", meta.GetOptions{})
	if err != nil {
		t.Errorf("error getting service monitor service: %v", err)
		return
	}

	// Verify that the endpoit is actually working
	metricsURL, err := url.Parse(getMetricsEndpointPath())
	if err != nil {
		t.Errorf("error parsing url for metrics: %v", err)
		return
	}

	if _, err := pkgTest.WaitForEndpointState(
		context.Background(),
		&pkgTest.KubeClient{Kube: ctx.Clients.Kube},
		t.Logf,
		metricsURL,
		pkgTest.EventuallyMatchesBody("# TYPE serverless_telemetry gauge"),
		"WaitForMetricsToServeText",
		true); err != nil {
		t.Errorf("the metrics endpoint is not accessible: %v", err)
	}

	// Verify that service metric has the right value
	stats, err := fetchTelemetryMetrics()
	if err != nil {
		t.Errorf("failed to get telemetry metrics: %v", err)
	}

	// Serving installs by default kn-cli related resources which we need to count too
	if stats != nil {
		if stats.services != (tc.expectedMetricCount + 1) {
			t.Errorf("Got = %v, want: %v for number of services", stats.services, tc.expectedMetricCount)
		}
		if stats.configurations != (tc.expectedMetricCount + 1) {
			t.Errorf("Got = %v, want: %v for number of services", stats.configurations, tc.expectedMetricCount)
		}
		if stats.revisions != (tc.expectedMetricCount + 1) {
			t.Errorf("Got = %v, want: %v for number of services", stats.revisions, tc.expectedMetricCount)
		}
		if stats.routes != (tc.expectedMetricCount + 1) {
			t.Errorf("Got = %v, want: %v for number of services", stats.routes, tc.expectedMetricCount)
		}
	}

	// Verify the expected istio-proxy is really there
	podList, err := ctx.Clients.Kube.CoreV1().Pods(namespace).List(context.Background(), meta.ListOptions{LabelSelector: "serving.knative.dev/service=" + service.Name})
	if err != nil {
		t.Errorf("error listing pods: %v", err)
		return
	}

	if len(podList.Items) == 0 {
		t.Errorf("any pod for ksvc %q dos not found", service.Name)
		return
	}

	for _, pod := range podList.Items {
		istioProxyFound := false
		for _, container := range pod.Spec.Containers {
			if container.Name == "istio-proxy" {
				istioProxyFound = true
			}
		}

		if tc.expectIstioSidecar != istioProxyFound {
			if tc.expectIstioSidecar {
				t.Errorf("TestCase %s expects istio-proxy to be present, but no such container exists in %s", tc.name, pod.Name)
			} else {
				t.Errorf("TestCase %s does not expect istio-proxy to be present in pod %s, but it has one", tc.name, pod.Name)
			}
		}
	}
}
