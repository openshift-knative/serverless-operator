package knativekafkae2e

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	sourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
)

func KnativeSourceChannelKnativeService(t *testing.T, createChannelOrFail func(*test.Context) duckv1.KReference) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.Eventing.MessagingV1().Subscriptions(test.Namespace).Delete(context.Background(), subscriptionName, metav1.DeleteOptions{})
		client.Clients.Eventing.SourcesV1().PingSources(test.Namespace).Delete(context.Background(), pingSourceName, metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	eventStore, ksvc := DeployKsvcWithEventInfoStoreOrFail(client, t, test.Namespace, helloWorldService)

	channelRef := createChannelOrFail(client)

	subscription := &messagingv1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subscriptionName,
			Namespace: test.Namespace,
		},
		Spec: messagingv1.SubscriptionSpec{
			Channel: channelRef,
			Subscriber: &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: ksvcAPIVersion,
					Kind:       ksvcKind,
					Name:       ksvc.Name,
				},
			},
		},
	}
	_, err := client.Clients.Eventing.MessagingV1().Subscriptions(test.Namespace).Create(context.Background(), subscription, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Subscription: ", err)
	}
	ps := &sourcesv1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: test.Namespace,
		},
		Spec: sourcesv1.PingSourceSpec{
			Data: PingSourceData,
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &channelRef,
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
