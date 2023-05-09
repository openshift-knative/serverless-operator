package knativekafkae2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kafkachannelv1beta1 "knative.dev/eventing-kafka/pkg/apis/messaging/v1beta1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	kafkaChannelName  = "smoke-kc"
	channelAPIVersion = "messaging.knative.dev/v1beta1"
	kafkaChannelKind  = "KafkaChannel"
	serviceAccount    = "default"
)

var (
	kafkaChannel = kafkachannelv1beta1.KafkaChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaChannelName,
			Namespace: test.Namespace,
		},
		Spec: kafkachannelv1beta1.KafkaChannelSpec{
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}
)

func TestSourceToKafkaChanelToKnativeService(t *testing.T) {
	KnativeSourceChannelKnativeService(t, func(client *test.Context) duckv1.KReference {
		channel, err := client.Clients.Kafka.MessagingV1beta1().KafkaChannels(test.Namespace).Create(context.Background(), &kafkaChannel, metav1.CreateOptions{})
		if err != nil {
			t.Fatal("Unable to create Channel: ", err)
		}

		client.AddToCleanup(func() error {
			return client.Clients.Kafka.MessagingV1beta1().KafkaChannels(test.Namespace).Delete(context.Background(), kafkaChannelName, metav1.DeleteOptions{})
		})

		return duckv1.KReference{
			APIVersion: channelAPIVersion,
			Kind:       kafkaChannelKind,
			Name:       channel.Name,
		}
	})
}
