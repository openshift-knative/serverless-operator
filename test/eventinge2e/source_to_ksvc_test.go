package eventinge2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestKnativeSourceToKnativeService(t *testing.T) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.Eventing.SourcesV1().PingSources(test.Namespace).Delete(context.Background(), pingSourceName, metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	eventStore, ksvc := DeployKsvcWithEventInfoStoreOrFail(client, t, test.Namespace, helloWorldService)

	ps := &sourcesv1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: test.Namespace,
		},
		Spec: sourcesv1.PingSourceSpec{
			Data: pingSourceData,
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
	_, err := client.Clients.Eventing.SourcesV1().PingSources(test.Namespace).Create(context.Background(), ps, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Knative PingSource not created: %+V", err)
	}

	AssertPingSourceDataReceivedAtLeastOnce(eventStore)
}
