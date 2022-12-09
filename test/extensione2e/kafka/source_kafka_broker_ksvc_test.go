package knativekafkae2e

import (
	"context"
	"fmt"
	"github.com/openshift-knative/serverless-operator/test/eventinge2e"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
)

const (
	kafkaChannelBrokerName            = "smoke-test-kafka-kafka-channel-broker"
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
			"channel-template-spec": fmt.Sprintf(`
apiVersion: %q
kind: %q`, channelAPIVersion, kafkaChannelKind),
		},
	}

	kafkaChannelBroker = &eventingv1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaChannelBrokerName,
			Namespace: test.Namespace,
		},
		Spec: eventingv1.BrokerSpec{
			Config: &duckv1.KReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       kafkaChannelTemplateConfigMapName,
			},
		},
	}
)

func TestSourceToKafkaChannelBasedBrokerToKnativeService(t *testing.T) {
	eventinge2e.KnativeSourceBrokerTriggerKnativeService(t, func(client *test.Context) *eventingv1.Broker {
		// Create the KafkaChannel template ConfigMap for the Broker
		_, err := client.Clients.Kube.CoreV1().ConfigMaps(test.Namespace).Create(context.Background(), kafkaChannelTemplateConfigMap, metav1.CreateOptions{})
		if err != nil {
			t.Fatal("Unable to create ConfigMap: ", err)
		}

		client.AddToCleanup(func() error {
			return client.Clients.Kube.CoreV1().ConfigMaps(test.Namespace).Delete(context.Background(), kafkaChannelTemplateConfigMapName, metav1.DeleteOptions{})
		})

		// Create the (kafka backed) kafkaChannelBroker
		broker, err := client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Create(context.Background(), kafkaChannelBroker, metav1.CreateOptions{})
		if err != nil {
			t.Fatal("Unable to create Kafka Backed Broker: ", err)
		}

		client.AddToCleanup(func() error {
			return client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Delete(context.Background(), kafkaChannelBrokerName, metav1.DeleteOptions{})
		})

		return broker
	})
}
