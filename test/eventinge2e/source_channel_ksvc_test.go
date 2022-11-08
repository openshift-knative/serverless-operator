package eventinge2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgTest "knative.dev/pkg/test"

	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	sourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	channelName       = "smoke-test-channel"
	subscriptionName  = "smoke-test-subscription"
	channelAPIVersion = "messaging.knative.dev/v1"
	channelKind       = "Channel"
)

func TestKnativeSourceChannelKnativeService(t *testing.T) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.Eventing.MessagingV1().Subscriptions(test.Namespace).Delete(context.Background(), subscriptionName, metav1.DeleteOptions{})
		client.Clients.Eventing.MessagingV1().Channels(test.Namespace).Delete(context.Background(), channelName, metav1.DeleteOptions{})
		client.Clients.Eventing.SourcesV1().PingSources(test.Namespace).Delete(context.Background(), pingSourceName, metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	// Setup a knative service
	ksvc, err := test.WithServiceReady(client, helloWorldService, test.Namespace, pkgTest.ImagePath(test.HelloworldGoImg))
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	imc := &messagingv1.Channel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      channelName,
			Namespace: test.Namespace,
		},
	}
	channel, err := client.Clients.Eventing.MessagingV1().Channels(test.Namespace).Create(context.Background(), imc, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Channel: ", err)
	}
	subscription := &messagingv1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subscriptionName,
			Namespace: test.Namespace,
		},
		Spec: messagingv1.SubscriptionSpec{
			Channel: duckv1.KReference{
				APIVersion: channelAPIVersion,
				Kind:       channelKind,
				Name:       channel.Name,
			},
			Subscriber: &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: ksvcAPIVersion,
					Kind:       ksvcKind,
					Name:       helloWorldService,
				},
			},
		},
	}
	_, err = client.Clients.Eventing.MessagingV1().Subscriptions(test.Namespace).Create(context.Background(), subscription, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Subscription: ", err)
	}
	ps := &sourcesv1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: test.Namespace,
		},
		Spec: sourcesv1.PingSourceSpec{
			Data: helloWorldText,
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: channelAPIVersion,
						Kind:       channelKind,
						Name:       channel.Name,
					},
				},
			},
		},
	}
	_, err = client.Clients.Eventing.SourcesV1().PingSources(test.Namespace).Create(context.Background(), ps, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Knative PingSource not created: %+V", err)
	}
	servinge2e.WaitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)
}
