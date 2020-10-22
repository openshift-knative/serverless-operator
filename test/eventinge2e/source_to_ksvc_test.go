package eventinge2e

import (
	"net/url"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingsourcesv1beta1 "knative.dev/eventing/pkg/apis/sources/v1beta1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	pkgTest "knative.dev/pkg/test"
)

const (
	pingSourceName    = "smoke-test-ping"
	testNamespace     = "serverless-tests"
	image             = "gcr.io/knative-samples/helloworld-go"
	helloWorldService = "helloworld-go"
	helloWorldText    = "Hello World!"
	ksvcAPIVersion    = "serving.knative.dev/v1"
	ksvcKind          = "Service"
)

func TestKnativeSourceToKnativeService(t *testing.T) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.Eventing.SourcesV1beta1().PingSources(testNamespace).Delete(pingSourceName, &metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer test.CleanupAll(t, client)
	defer cleanup()

	// Setup a knative service
	ksvc, err := test.WithServiceReady(client, helloWorldService, testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	ps := &eventingsourcesv1beta1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: testNamespace,
		},
		Spec: eventingsourcesv1beta1.PingSourceSpec{
			JsonData: helloWorldText,
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: ksvcAPIVersion,
						Kind:       ksvcKind,
						Name:       ksvc.Name,
					},
				},
			},
		},
	}
	_, err = client.Clients.Eventing.SourcesV1beta1().PingSources(testNamespace).Create(ps)
	if err != nil {
		t.Fatal("Knative PingSource not created: %+V", err)
	}
	waitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)

	// Delete the PingSource
	client.Clients.Eventing.SourcesV1beta1().PingSources(testNamespace).Delete(ps.Name, &metav1.DeleteOptions{})
}

func waitForRouteServingText(t *testing.T, client *test.Context, routeURL *url.URL, expectedText string) {
	t.Helper()
	if _, err := pkgTest.WaitForEndpointState(
		&pkgTest.KubeClient{Kube: client.Clients.Kube},
		t.Logf,
		routeURL,
		pkgTest.EventuallyMatchesBody(expectedText),
		"WaitForRouteToServeText",
		true); err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected text \"%s\": %v", routeURL, expectedText, err)
	}

}
