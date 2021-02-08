package servinge2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress/resources"
	"github.com/openshift-knative/serverless-operator/test"
	routev1 "github.com/openshift/api/route/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

// TestCustomOpenShiftRoute verifies user-defined OpenShift Route could work.
// 1. Create Kservice with disableRoute annotation.
// 2. Verify operator did not create OpenShift Route.
// 3. Create the OpenShift Route manually.
// 4. Verify the access.
func TestCustomOpenShiftRoute(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	// Create Kservice with disable Annotation.
	ksvc := test.Service("custom-route-test", testNamespace, image, nil)
	ksvc.ObjectMeta.Annotations = map[string]string{resources.DisableRouteAnnotation: "true"}
	ksvc = withServiceReadyOrFail(caCtx, ksvc)

	// Verify that operator did not create OpenShift route.
	routes, err := caCtx.Clients.Route.Routes("knative-serving-ingress").List(context.Background(), meta.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", resources.OpenShiftIngressLabelKey, ksvc.Name),
	})
	if err != nil {
		t.Fatalf("Failed to list routes: %v", err)
	}
	if len(routes.Items) != 0 {
		t.Fatalf("Unexpected routes found: %v", routes)
	}

	// Create OpenShift Route manually
	route := &routev1.Route{
		ObjectMeta: meta.ObjectMeta{
			Name:      "myroute",
			Namespace: "knative-serving-ingress",
		},
		Spec: routev1.RouteSpec{
			Host: ksvc.Status.URL.Host,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(resources.KourierHTTPPort),
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "kourier",
			},
		},
	}
	route, err = caCtx.Clients.Route.Routes("knative-serving-ingress").Create(context.Background(), route, meta.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating OpenShift Route: %v", err)
	}

	caCtx.AddToCleanup(func() error {
		t.Logf("Cleaning up OpenShift Route %s", route.Name)
		return caCtx.Clients.Route.Routes(route.Namespace).Delete(context.Background(), route.Name, meta.DeleteOptions{})
	})

	// Retry until OpenShift Route becomes ready.
	err = wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
		resp, inErr := http.Get(ksvc.Status.URL.String())
		if inErr != nil {
			return false, inErr
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Logf("Retrying... route might not be ready yet")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to verify custom route: %v", err)
	}

}
