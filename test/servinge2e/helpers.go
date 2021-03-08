package servinge2e

import (
	"context"
	"net/url"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgTest "knative.dev/pkg/test"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1alpha1 "knative.dev/serving/pkg/apis/serving/v1alpha1"
)

func WaitForRouteServingText(t *testing.T, caCtx *test.Context, routeURL *url.URL, expectedText string) {
	t.Helper()
	if _, err := pkgTest.WaitForEndpointState(
		context.Background(),
		caCtx.Clients.Kube,
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

func withDomainMappingReadyOrFail(ctx *test.Context, dm *servingv1alpha1.DomainMapping) *servingv1alpha1.DomainMapping {
	dm, err := ctx.Clients.Serving.ServingV1alpha1().DomainMappings(dm.Namespace).Create(context.Background(), dm, meta.CreateOptions{})
	if err != nil {
		ctx.T.Fatalf("Error creating ksvc: %v", err)
	}

	// Let the ksvc be deleted after test
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Knative Service '%s/%s'", dm.Namespace, dm.Name)
		return ctx.Clients.Serving.ServingV1alpha1().DomainMappings(dm.Namespace).Delete(context.Background(), dm.Name, meta.DeleteOptions{})
	})

	dm, err = test.WaitForDomainMappingState(ctx, dm.Name, dm.Namespace, test.IsDomainMappingReady)
	if err != nil {
		ctx.T.Fatalf("Error waiting for ksvc readiness: %v", err)
	}

	return dm
}
