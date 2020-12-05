package eventinge2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingmessagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	eventingsourcesv1beta1 "knative.dev/eventing/pkg/apis/sources/v1beta1"
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
		client.Clients.Eventing.MessagingV1().Subscriptions(testNamespace).Delete(context.Background(), subscriptionName, metav1.DeleteOptions{})
		client.Clients.Eventing.MessagingV1().Channels(testNamespace).Delete(context.Background(), channelName, metav1.DeleteOptions{})
		client.Clients.Eventing.SourcesV1beta1().PingSources(testNamespace).Delete(context.Background(), pingSourceName, metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	// Setup a knative service
	ksvc, err := test.WithServiceReady(client, helloWorldService, testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	imc := &eventingmessagingv1.Channel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      channelName,
			Namespace: testNamespace,
		},
	}
	channel, err := client.Clients.Eventing.MessagingV1().Channels(testNamespace).Create(context.Background(), imc, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Channel: ", err)
	}
	subscription := &eventingmessagingv1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subscriptionName,
			Namespace: testNamespace,
		},
		Spec: eventingmessagingv1.SubscriptionSpec{
			Channel: corev1.ObjectReference{
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
	_, err = client.Clients.Eventing.MessagingV1().Subscriptions(testNamespace).Create(context.Background(), subscription, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Subscription: ", err)
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
						APIVersion: channelAPIVersion,
						Kind:       channelKind,
						Name:       channel.Name,
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
