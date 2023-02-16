package eventinge2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	brokerName = "smoke-test-broker"
)

func TestKnativeSourceBrokerTriggerKnativeService(t *testing.T) {
	KnativeSourceBrokerTriggerKnativeService(t, func(client *test.Context) *eventingv1.Broker {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: cmName,
			},
			Data: map[string]string{
				"channel-template-spec": fmt.Sprintf(`
apiVersion: %q
kind: %q`, channelAPIVersion, channelKind),
			},
		}
		configMap, err := client.Clients.Kube.CoreV1().ConfigMaps(test.Namespace).Create(context.Background(), cm, metav1.CreateOptions{})
		if err != nil {
			t.Fatal("Unable to create ConfigMap: ", err)
		}

		client.AddToCleanup(func() error {
			return client.Clients.Kube.CoreV1().ConfigMaps(test.Namespace).Delete(context.Background(), cmName, metav1.DeleteOptions{})
		})

		br := &eventingv1.Broker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      brokerName,
				Namespace: test.Namespace,
			},
			Spec: eventingv1.BrokerSpec{
				Config: &duckv1.KReference{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       configMap.Name,
				},
			},
		}
		broker, err := client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Create(context.Background(), br, metav1.CreateOptions{})
		if err != nil {
			t.Fatal("Unable to create broker: ", err)
		}

		client.AddToCleanup(func() error {
			return client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Delete(context.Background(), brokerName, metav1.DeleteOptions{})
		})

		return broker
	}, nil)
}
