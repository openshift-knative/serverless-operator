package eventinge2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	channelName       = "smoke-test-channel"
	channelAPIVersion = "messaging.knative.dev/v1"
	channelKind       = "Channel"
)

func TestKnativeSourceChannelKnativeService(t *testing.T) {
	KnativeSourceChannelKnativeService(t, func(client *test.Context) duckv1.KReference {
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

		client.AddToCleanup(func() error {
			return client.Clients.Eventing.MessagingV1().Channels(test.Namespace).Delete(context.Background(), channelName, metav1.DeleteOptions{})
		})

		return duckv1.KReference{
			APIVersion: channelAPIVersion,
			Kind:       channelKind,
			Name:       channel.Name,
		}
	})
}
