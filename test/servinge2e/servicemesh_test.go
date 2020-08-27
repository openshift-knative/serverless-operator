package servinge2e

import (
	"github.com/openshift-knative/serverless-operator/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgTest "knative.dev/pkg/test"
	"testing"
)

const (
	// A test namespace that is part of the ServiceMesh (setup by "make install-mesh")
	serviceMeshTestNamespaceName = "default"
	serviceMeshTestImage = "gcr.io/knative-samples/helloworld-go"
)

// Skipped unless ServiceMesh has been installed via "make install-mesh"
func TestKsvcWithServiceMeshSidecar(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	// Verify the serviceMeshTestNamespace is part of the mesh
	namespace, err := caCtx.Clients.Kube.CoreV1().Namespaces().Get(serviceMeshTestNamespaceName, meta.GetOptions{})
	if err != nil {
		t.Fatalf("failed to verify %q namespace labels: %v", serviceMeshTestNamespaceName, err)
	}

	if namespace.Labels["maistra.io/member-of"] == "" {
		t.Skipf("test namespace %q not a mesh member, use \"make install-mesh\" for ServiceMesh setup", serviceMeshTestNamespaceName)
	}

	tests := []struct {
		name string
		annotations map[string]string
		expectIstioSidecar bool
	}{{
		// Requests go via gateway -> activator -> pod , by default
		// Verifies the activator can connect to the pod
		name: "sidecar-via-activator",
		annotations: map[string]string{
			"sidecar.istio.io/inject":          "true",
		},
		expectIstioSidecar: true,
	},{
		// Requests go via gateway -> pod ( activator should be skipped if burst capacity is disabled and there is at least 1 replica)
		// Verifies the gateway can connect to the pod directly
		name: "sidecar-without-activator",
		annotations: map[string]string{
			"sidecar.istio.io/inject":											"true",
			"autoscaling.knative.dev/targetBurstCapacity": 	"0",
			"autoscaling.knative.dev/minScale": 						"1",
		},
		expectIstioSidecar: true,
	},{
		// Verifies the "sidecar.istio.io/inject" annotation is really what decides the istio-proxy presence
		name: "no-sidecar",
		annotations: map[string]string{
			"sidecar.istio.io/inject":					"false",
		},
		expectIstioSidecar: false,
	}}

	for _, scenario := range tests {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			// Create a ksvc with the specified annotations
			service, err := caCtx.Clients.Serving.ServingV1().Services(serviceMeshTestNamespaceName).Create(
				test.Service(scenario.name, serviceMeshTestNamespaceName, serviceMeshTestImage, scenario.annotations))
			if err != nil {
				t.Errorf("error creating ksvc: %v", err)
				return
			}

			// Let the ksvc be deleted after test
			caCtx.AddToCleanup(func() error {
				t.Logf("Cleaning up Knative Service '%s/%s'", service.Namespace, service.Name)
				return caCtx.Clients.Serving.ServingV1().Services(serviceMeshTestNamespaceName).Delete(service.Name, &meta.DeleteOptions{})
			})

			// Wait until the Ksvc is ready.
			service, err = test.WaitForServiceState(caCtx, service.Name, service.Namespace, test.IsServiceReady)
			if err != nil {
				t.Errorf("error waiting for ksvc readiness: %v", err)
				return
			}

			// Verify the service is actually accessible from the outside
			if _, err := pkgTest.WaitForEndpointState(
				&pkgTest.KubeClient{Kube: caCtx.Clients.Kube},
				t.Logf,
				service.Status.URL.URL(),
				pkgTest.EventuallyMatchesBody(helloworldText),
				"WaitForRouteToServeText",
				true); err != nil {
				t.Errorf("the Route at domain %s didn't serve the expected text %q: %v", service.Status.URL.URL(), helloworldText, err)
			}

			// Verify the expected istio-proxy is really there
			podList, err := caCtx.Clients.Kube.CoreV1().Pods(serviceMeshTestNamespaceName).List(meta.ListOptions{LabelSelector: "serving.knative.dev/service=" + service.Name})
			if err != nil {
				t.Errorf("error listing pods: %v", err)
				return
			}

			for _, pod := range podList.Items {
				istioProxyFound := false
				for _, container := range pod.Spec.Containers {
					if container.Name == "istio-proxy" {
						istioProxyFound = true
					}
				}

				if scenario.expectIstioSidecar != istioProxyFound {
					if scenario.expectIstioSidecar {
						t.Errorf("scenario %s expects istio-proxy to be present, but no such container exists in %s", scenario.name, pod.Name)
					} else {
						t.Errorf("scenario %s does not expect istio-proxy to be present in pod %s, but it has one", scenario.name, pod.Name)
					}
				}
			}
		})
	}
}
