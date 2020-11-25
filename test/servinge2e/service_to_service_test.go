package servinge2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	network "knative.dev/networking/pkg"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/serving/pkg/apis/autoscaling"
	"knative.dev/serving/pkg/apis/serving"
)

// Smoke tests for networking which access public and cluster-local
// services from within the cluster.
func TestServiceToServiceCalls(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	tests := []testCase{{
		// Requests go via gateway -> activator -> pod.
		name: "service-call-via-activator",
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "-1",
		},
	}, {
		// Requests go via gateway -> pod (activator should be skipped if burst
		// capacity is disabled and there is at least 1 replica).
		name: "service-call-without-activator",
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "0",
			autoscaling.MinScaleAnnotationKey:  "1",
		},
	}, {
		name: "cluster-local-via-activator",
		labels: map[string]string{
			network.VisibilityLabelKey: serving.VisibilityClusterLocal,
		},
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "-1",
		},
	}, {
		name: "cluster-local-without-activator",
		labels: map[string]string{
			network.VisibilityLabelKey: serving.VisibilityClusterLocal,
		},
		annotations: map[string]string{
			autoscaling.TargetBurstCapacityKey: "0",
			autoscaling.MinScaleAnnotationKey:  "1",
		},
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
	if service.GetLabels()[network.VisibilityLabelKey] == serving.VisibilityClusterLocal {
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
