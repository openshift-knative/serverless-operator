package knativekafkae2e

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	sourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
)

const (
	kafkaChannelBrokerName            = "smoke-test-kafka-kafka-channel-broker"
	kafkatriggerName                  = "smoke-test-trigger"
	kafkaChannelTemplateConfigMapName = "smoke-test-br-cm"
	brokerAPIVersion                  = "eventing.knative.dev/v1"
	brokerKind                        = "Broker"
)

var (
	kafkaChannelTemplateConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: kafkaChannelTemplateConfigMapName,
		},
		Data: map[string]string{
			"channelTemplateSpec": fmt.Sprintf(`
apiVersion: %q
kind: %q`, channelAPIVersion, kafkaChannelKind),
		},
	}

	kafkaChannelBroker = &eventingv1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaChannelBrokerName,
			Namespace: testNamespace,
		},
		Spec: eventingv1.BrokerSpec{
			Config: &duckv1.KReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       kafkaChannelTemplateConfigMapName,
			},
		},
	}

	trigger = &eventingv1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkatriggerName,
			Namespace: testNamespace,
		},
		Spec: eventingv1.TriggerSpec{
			Broker: kafkaChannelBrokerName,
			Subscriber: duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: ksvcAPIVersion,
					Kind:       ksvcKind,
					Name:       helloWorldService + "-kafka-channel-broker",
				},
			},
		},
	}

	brokerPingSource = &sourcesv1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: testNamespace,
		},
		Spec: sourcesv1.PingSourceSpec{
			Data: helloWorldText,
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: brokerAPIVersion,
						Kind:       brokerKind,
						Name:       kafkaChannelBrokerName,
					},
				},
			},
		},
	}
)

func TestSourceToKafkaChannelBasedBrokerToKnativeService(t *testing.T) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.Eventing.EventingV1().Brokers(testNamespace).Delete(context.Background(), kafkaChannelBrokerName, metav1.DeleteOptions{})
		client.Clients.Eventing.SourcesV1().PingSources(testNamespace).Delete(context.Background(), pingSourceName, metav1.DeleteOptions{})
		client.Clients.Eventing.EventingV1().Triggers(testNamespace).Delete(context.Background(), kafkatriggerName, metav1.DeleteOptions{})
		client.Clients.Kube.CoreV1().ConfigMaps(testNamespace).Delete(context.Background(), kafkaChannelTemplateConfigMapName, metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	ksvc, err := test.WithServiceReady(client, helloWorldService+"-kafka-channel-broker", testNamespace, image, nil)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Create the configmap
	_, err = client.Clients.Kube.CoreV1().ConfigMaps(testNamespace).Create(context.Background(), kafkaChannelTemplateConfigMap, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Channel Template ConfigMap: ", err)
	}

	// Create the (kafka backed) kafkaChannelBroker
	_, err = client.Clients.Eventing.EventingV1().Brokers(testNamespace).Create(context.Background(), kafkaChannelBroker, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Kafka Backed Broker: ", err)
	}

	// Create the Trigger
	_, err = client.Clients.Eventing.EventingV1().Triggers(testNamespace).Create(context.Background(), trigger, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create trigger: ", err)
	}

	// Create the source
	_, err = client.Clients.Eventing.SourcesV1().PingSources(testNamespace).Create(context.Background(), brokerPingSource, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create pingsource: ", err)
	}

	// Wait for text in kservice
	servinge2e.WaitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)
}
