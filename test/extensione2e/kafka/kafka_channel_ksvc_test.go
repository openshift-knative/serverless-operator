package knativekafkae2e

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kafkachannelv1beta1 "knative.dev/eventing-kafka/pkg/apis/messaging/v1beta1"
	eventingmessagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	eventingsourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
)

const (
	pingSourceName    = "smoke-test-pingsource"
	helloWorldText    = "Hello World!"
	kafkaChannelName  = "smoke-kc"
	channelAPIVersion = "messaging.knative.dev/v1beta1"
	kafkaChannelKind  = "KafkaChannel"
	subscriptionName  = "smoke-test-kafka-subscription"
	serviceAccount    = "default"
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

	subscription = &eventingmessagingv1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subscriptionName,
			Namespace: testNamespace,
		},
		Spec: eventingmessagingv1.SubscriptionSpec{
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

	ps = &eventingsourcesv1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: testNamespace,
		},
		Spec: eventingsourcesv1.PingSourceSpec{
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
		client.Clients.Kafka.MessagingV1beta1().KafkaChannels(testNamespace).Delete(context.Background(), kafkaChannelName, metav1.DeleteOptions{})
		client.Clients.Eventing.MessagingV1().Subscriptions(testNamespace).Delete(context.Background(), subscriptionName, metav1.DeleteOptions{})
		client.Clients.Eventing.SourcesV1beta1().PingSources(testNamespace).Delete(context.Background(), pingSourceName, metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	// Setup a knative service
	ksvc, err := test.WithServiceReady(client, helloWorldService, testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Create kafka channel
	_, err = client.Clients.Kafka.MessagingV1beta1().KafkaChannels(testNamespace).Create(context.Background(), &kafkaChannel, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create KafkaChannel: ", err)
	}

	// Create subscription (from channel to service)
	_, err = client.Clients.Eventing.MessagingV1().Subscriptions(testNamespace).Create(context.Background(), subscription, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Subscription: ", err)
	}

	// Create source (channel as sink)
	_, err = client.Clients.Eventing.SourcesV1beta1().PingSources(testNamespace).Create(context.Background(), ps, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Knative PingSource not created: ", err)
	}

	servinge2e.WaitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)
}
