package knativekafkae2e

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kafkachannelv1beta1 "knative.dev/eventing-contrib/kafka/channel/pkg/apis/messaging/v1beta1"
	eventingmessagingv1beta1 "knative.dev/eventing/pkg/apis/messaging/v1beta1"
	eventingsourcesv1alpha2 "knative.dev/eventing/pkg/apis/sources/v1alpha2"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
)

const (
	pingSourceName    = "smoke-test-pingsource"
	helloWorldText    = "Hello World!"
	kafkaChannelName  = "smoke-kc"
	channelAPIVersion = "messaging.knative.dev/v1beta1"
	kafkaChannelKind  = "KafkaChannel"
	subscriptionName  = "smoke-test-kafka-subscription"
)

var (
	kafkaChannel = kafkachannelv1beta1.KafkaChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaChannelName,
			Namespace: testNamespace,
		},
		Spec: kafkachannelv1beta1.KafkaChannelSpec{
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	subscription = &eventingmessagingv1beta1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subscriptionName,
			Namespace: testNamespace,
		},
		Spec: eventingmessagingv1beta1.SubscriptionSpec{
			Channel: corev1.ObjectReference{
				APIVersion: channelAPIVersion,
				Kind:       kafkaChannelKind,
				Name:       kafkaChannelName,
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

	ps = &eventingsourcesv1alpha2.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: testNamespace,
		},
		Spec: eventingsourcesv1alpha2.PingSourceSpec{
			JsonData: helloWorldText,
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: channelAPIVersion,
						Kind:       kafkaChannelKind,
						Name:       kafkaChannelName,
					},
				},
			},
		},
	}
)

func TestSourceToKafkaChanelToKnativeService(t *testing.T) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.KafkaChannel.MessagingV1beta1().KafkaChannels(testNamespace).Delete(kafkaChannelName, &metav1.DeleteOptions{})
		client.Clients.Eventing.MessagingV1beta1().Subscriptions(testNamespace).Delete(subscriptionName, &metav1.DeleteOptions{})
		client.Clients.Eventing.SourcesV1alpha2().PingSources(testNamespace).Delete(pingSourceName, &metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer test.CleanupAll(t, client)
	defer cleanup()

	// Setup a knative service
	ksvc, err := test.WithServiceReady(client, helloWorldService, testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Create kafka channel
	_, err = client.Clients.KafkaChannel.MessagingV1beta1().KafkaChannels(testNamespace).Create(&kafkaChannel)
	if err != nil {
		t.Fatal("Unable to create KafkaChannel: ", err)
	}

	// Create subscription (from channel to service)
	_, err = client.Clients.Eventing.MessagingV1beta1().Subscriptions(testNamespace).Create(subscription)
	if err != nil {
		t.Fatal("Unable to create Subscription: ", err)
	}

	// Create source (channel as sink)
	_, err = client.Clients.Eventing.SourcesV1alpha2().PingSources(testNamespace).Create(ps)
	if err != nil {
		t.Fatal("Knative PingSource not created: ", err)
	}

	waitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)

	// cleanup if everything ends smoothly
	cleanup()
}
