package eventinge2e

import (
	"context"
	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	sourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"testing"
)

func KnativeSourceBrokerTriggerKnativeService(t *testing.T, createBrokerOrFail func(*test.Context) *eventingv1.Broker) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.Eventing.EventingV1().Triggers(test.Namespace).Delete(context.Background(), triggerName, metav1.DeleteOptions{})
		client.Clients.Eventing.SourcesV1().PingSources(test.Namespace).Delete(context.Background(), pingSourceName, metav1.DeleteOptions{})
		client.Clients.Kube.CoreV1().ConfigMaps(test.Namespace).Delete(context.Background(), cmName, metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	eventStore, ksvc := DeployKsvcWithEventInfoStoreOrFail(client, t, test.Namespace, helloWorldService)

	broker := createBrokerOrFail(client)

	tr := &eventingv1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: test.Namespace,
		},
		Spec: eventingv1.TriggerSpec{
			Broker: broker.Name,
			Subscriber: duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: ksvcAPIVersion,
					Kind:       ksvcKind,
					Name:       ksvc.Name,
				},
			},
		},
	}
	_, err := client.Clients.Eventing.EventingV1().Triggers(test.Namespace).Create(context.Background(), tr, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create trigger: ", err)
	}

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
						APIVersion: brokerAPIVersion,
						Kind:       brokerKind,
						Name:       broker.Name,
					},
				},
			},
		},
	}
	_, err = client.Clients.Eventing.SourcesV1().PingSources(test.Namespace).Create(context.Background(), ps, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Knative PingSource not created: %+V", err)
	}

	AssertPingSourceDataReceivedAtLeastOnce(eventStore)
}
