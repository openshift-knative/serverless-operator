package servinge2e

import (
	"context"
	"net/url"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestKnativeVersusKubeServicesInOneNamespace(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	//Create deployment
	err := test.CreateDeployment(caCtx, kubeHelloworldService, testNamespace2, image)
	if err != nil {
		t.Fatal("Deployment not created", err)
	}
	// Deploy plain Kube service
	svc, err := createKubeService(caCtx, kubeHelloworldService, testNamespace2)
	if err != nil {
		t.Fatal("Kubernetes service not created", err)
	}
	route, err := withRouteForServiceReady(caCtx, svc.Name, testNamespace2)
	if err != nil {
		t.Fatal("Failed to create route for service", svc.Name, err)
	}
	kubeServiceURL, err := url.Parse("http://" + route.Status.Ingress[0].Host)
	if err != nil {
		t.Fatal("Failed to parse url", err)
	}

	// Check Kube service responds
	WaitForRouteServingText(t, caCtx, kubeServiceURL, helloworldText)

	// Deploy Knative service in the same namespace
	ksvc, err := test.WithServiceReady(caCtx, helloworldService, testNamespace2, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Check that both services respond
	WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), helloworldText)
	WaitForRouteServingText(t, caCtx, kubeServiceURL, helloworldText)

	// Delete Knative service
	caCtx.Clients.Serving.ServingV1().Services(testNamespace2).Delete(context.Background(), ksvc.Name, metav1.DeleteOptions{})

	// Check that Kube service still responds
	WaitForRouteServingText(t, caCtx, kubeServiceURL, helloworldText)

	// Remove the Kube service
	caCtx.Clients.Route.Routes(testNamespace2).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})
	caCtx.Clients.Kube.CoreV1().Services(testNamespace2).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})
	caCtx.Clients.Kube.AppsV1().Deployments(testNamespace2).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})

	// Deploy Knative service in the namespace first
	ksvc, err = test.WithServiceReady(caCtx, helloworldService2, testNamespace2, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Check that Knative service responds
	WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), helloworldText)

	//Create deployment
	err = test.CreateDeployment(caCtx, kubeHelloworldService, testNamespace2, image)
	if err != nil {
		t.Fatal("Deployment not created", err)
	}
	// Deploy plain Kube service
	svc, err = createKubeService(caCtx, kubeHelloworldService, testNamespace2)
	if err != nil {
		t.Fatal("Kubernetes service not created", err)
	}
	route, err = withRouteForServiceReady(caCtx, svc.Name, testNamespace2)
	if err != nil {
		t.Fatal("Failed to create route for service", svc.Name, err)
	}
	kubeServiceURL, err = url.Parse("http://" + route.Status.Ingress[0].Host)
	if err != nil {
		t.Fatal("Failed to parse url", err)
	}

	// Check that both services respond
	WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), helloworldText)
	WaitForRouteServingText(t, caCtx, kubeServiceURL, helloworldText)

	// Remove the Kube service
	caCtx.Clients.Route.Routes(testNamespace2).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})
	caCtx.Clients.Kube.CoreV1().Services(testNamespace2).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})
	caCtx.Clients.Kube.AppsV1().Deployments(testNamespace2).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})

	// Check that Knative service still responds
	WaitForRouteServingText(t, caCtx, ksvc.Status.URL.URL(), helloworldText)

	// Delete the Knative service
	caCtx.Clients.Serving.ServingV1().Services(testNamespace2).Delete(context.Background(), ksvc.Name, metav1.DeleteOptions{})
}

func withRouteForServiceReady(ctx *test.Context, serviceName, namespace string) (*routev1.Route, error) {
	r := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: serviceName,
			},
		},
	}

	route, err := ctx.Clients.Route.Routes(namespace).Create(context.Background(), r, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up OCP Route '%s/%s'", r.Namespace, r.Name)
		return ctx.Clients.Route.Routes(namespace).Delete(context.Background(), route.Name, metav1.DeleteOptions{})
	})

	return test.WaitForRouteState(ctx, route.Name, route.Namespace, routeHasHost)
}

func routeHasHost(r *routev1.Route, err error) (bool, error) {
	return len(r.Status.Ingress) != 0 && len(r.Status.Ingress[0].Conditions) != 0 &&
		r.Status.Ingress[0].Conditions[0].Type == routev1.RouteAdmitted &&
		r.Status.Ingress[0].Conditions[0].Status == corev1.ConditionTrue, nil
}

func createKubeService(ctx *test.Context, name, namespace string) (*corev1.Service, error) {
	kubeService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8080,
					},
				},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}

	svc, err := ctx.Clients.Kube.CoreV1().Services(namespace).Create(context.Background(), kubeService, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up K8s Service '%s/%s'", kubeService.Namespace, kubeService.Name)
		return ctx.Clients.Serving.ServingV1().Services(namespace).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})
	})

	return svc, nil
}
