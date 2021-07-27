package kourier

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress/resources"
	"github.com/openshift-knative/serverless-operator/test"
	routev1 "github.com/openshift/api/route/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/test/spoof"
	servingv1alpha1 "knative.dev/serving/pkg/apis/serving/v1alpha1"
)

const (
	serviceName       = "custom-route-test"
	domainMappingName = "mydomain-test.example.com"
)

// TestCustomOpenShiftRoute verifies user-defined OpenShift Route could work.
// 1. Create Kservice with disableRoute annotation.
// 2. Verify operator did not create OpenShift Route.
// 3. Create the OpenShift Route manually.
// 4. Verify the access.
// 5. Create DomainMapping with disableRoute annotation.
// 6. Verify operator did not create OpenShift Route for domainmapping.
// 7. Create the OpenShift Route manually.
// 8. Verify the access for customdomain.
func TestCustomOpenShiftRoute(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	// Create Kservice with disable Annotation.
	ksvc := test.Service(serviceName, testNamespace, image, nil)
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
				TargetPort: intstr.FromString(resources.HTTPPort),
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

	// Create DomainMapping with disable Annotation.
	dm := &servingv1alpha1.DomainMapping{
		ObjectMeta: meta.ObjectMeta{
			Name:        domainMappingName,
			Namespace:   testNamespace,
			Annotations: map[string]string{resources.DisableRouteAnnotation: "true"},
		},
		Spec: servingv1alpha1.DomainMappingSpec{
			Ref: duckv1.KReference{
				Kind:       "Service",
				Name:       serviceName,
				APIVersion: "serving.knative.dev/v1",
			},
		},
	}

	dm = withDomainMappingReadyOrFail(caCtx, dm)

	// Verify that operator did not create OpenShift route.
	routes, err = caCtx.Clients.Route.Routes("knative-serving-ingress").List(context.Background(), meta.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", resources.OpenShiftIngressLabelKey, dm.Name),
	})
	if err != nil {
		t.Fatalf("Failed to list routes: %v", err)
	}
	if len(routes.Items) != 0 {
		t.Fatalf("Unexpected routes found: %v", routes)
	}

	// Create OpenShift Route manually
	route = &routev1.Route{
		ObjectMeta: meta.ObjectMeta{
			Name:      "myroute-for-dm",
			Namespace: "knative-serving-ingress",
		},
		Spec: routev1.RouteSpec{
			Host: dm.Status.URL.Host,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(resources.HTTPPort),
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

	routerIP := lookupOpenShiftRouterIP(caCtx)
	sc, err := newSpoofClientWithTLS(caCtx, domainMappingName, routerIP.String(), nil)
	if err != nil {
		t.Fatalf("Error creating a Spoofing Client: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "http://"+domainMappingName, nil)
	if err != nil {
		t.Fatalf("Error creating an HTTP GET request: %v", err)
	}

	// Retry until OpenShift Route becomes ready.
	resp, err := sc.Poll(req, spoof.IsStatusOK)
	if err != nil {
		t.Fatalf("Error polling custom domain: %v", err)
	}
	const expectedResponse = "Hello World!"
	if resp.StatusCode != 200 || strings.TrimSpace(string(resp.Body)) != expectedResponse {
		t.Fatalf("Expecting a HTTP 200 response with %q, got %d: %s", expectedResponse, resp.StatusCode, string(resp.Body))
	}
}
