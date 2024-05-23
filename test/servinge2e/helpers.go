package servinge2e

import (
	"context"
	"net/url"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/spoof"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingTest "knative.dev/serving/test"
)

const (
	HelloworldText = "Hello World!"
)

func WaitForRouteServingText(t *testing.T, caCtx *test.Context, routeURL *url.URL, expectedText string) {
	t.Helper()
	if _, err := pkgTest.CheckEndpointState(
		context.Background(),
		caCtx.Clients.Kube,
		t.Logf,
		routeURL,
		spoof.MatchesBody(expectedText),
		"WaitForRouteToServeText",
		true,
		servingTest.AddRootCAtoTransport(context.Background(), t.Logf, &servingTest.Clients{KubeClient: caCtx.Clients.Kube}, true),
	); err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected text %q: %v", routeURL, expectedText, err)
	}
}

func MakeSpoofingClient(ctx *test.Context, url *url.URL) (*spoof.SpoofingClient, error) {
	return pkgTest.NewSpoofingClient(
		context.Background(),
		ctx.Clients.Kube,
		ctx.T.Logf,
		url.Hostname(),
		true,
		servingTest.AddRootCAtoTransport(context.Background(), ctx.T.Logf, &servingTest.Clients{KubeClient: ctx.Clients.Kube}, true))
}

// HTTPProxyService returns a knative service acting as "http proxy", redirects requests towards a given "host". Used to test cluster-local services
func HTTPProxyService(name, namespace, gateway, target, cacert string, serviceAnnotations, templateAnnotations map[string]string) *servingv1.Service {
	proxy := test.Service(name, namespace, pkgTest.ImagePath(test.HTTPProxyImg), serviceAnnotations, templateAnnotations)
	if gateway != "" {
		proxy.Spec.Template.Spec.Containers[0].Env = append(proxy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "GATEWAY_HOST",
			Value: gateway,
		})
	}
	if cacert != "" {
		proxy.Spec.Template.Spec.Containers[0].Env = append(proxy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "CA_CERT",
			Value: cacert,
		})
	}
	proxy.Spec.Template.Spec.Containers[0].Env = append(proxy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "TARGET_HOST",
		Value: target,
	})

	return proxy
}
