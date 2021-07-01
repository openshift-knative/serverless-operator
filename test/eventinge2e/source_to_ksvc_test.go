package eventinge2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingsourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
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
		client.Clients.Eventing.SourcesV1beta1().PingSources(testNamespace).Delete(context.Background(), pingSourceName, metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	// Setup a knative service
	ksvc, err := test.WithServiceReady(client, helloWorldService, testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	ps := &eventingsourcesv1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: testNamespace,
		},
		Spec: eventingsourcesv1.PingSourceSpec{
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
	_, err = client.Clients.Eventing.SourcesV1beta1().PingSources(testNamespace).Create(context.Background(), ps, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Knative PingSource not created: %+V", err)
	}
	servinge2e.WaitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)
}
