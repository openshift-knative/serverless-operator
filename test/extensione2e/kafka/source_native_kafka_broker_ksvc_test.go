package knativekafkae2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/openshift-knative/serverless-operator/test/eventinge2e"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
)

const (
	nativeKafkaBrokerName = "smoke-test-native-kafka-broker"
)

var (
	nativeKafkaBroker = &eventingv1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nativeKafkaBrokerName,
			Namespace:   test.Namespace,
			Annotations: map[string]string{"eventing.knative.dev/broker.class": "Kafka"},
		},
		Spec: eventingv1.BrokerSpec{
			Config: &duckv1.KReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "kafka-broker-config",
				Namespace:  "knative-eventing",
			},
		},
	}
)

func TestSourceToNativeKafkaBasedBrokerToKnativeService(t *testing.T) {
	eventinge2e.KnativeSourceBrokerTriggerKnativeService(t, func(client *test.Context) *eventingv1.Broker {
		// Create the (native) Kafka Broker
		broker, err := client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Create(context.Background(), nativeKafkaBroker, metav1.CreateOptions{})
		if err != nil {
			t.Fatal("Unable to create Kafka Backed Broker: ", err)
		}

		client.AddToCleanup(func() error {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
			defer cancel()

			err := wait.PollImmediateUntil(2*time.Second, waitForBrokerDeletion(ctx, client, t), ctx.Done())
			if err != nil {
				t.Fatal(err)
			}

			cmName := nativeKafkaBroker.Spec.Config.Name
			cmNamepace := nativeKafkaBroker.Spec.Config.Namespace
			cm, err := client.Clients.Kube.
				CoreV1().
				ConfigMaps(cmNamepace).
				Get(ctx, cmName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("failed to get ConfigMap")
			}
			for _, f := range cm.GetFinalizers() {
				if strings.Contains(f, nativeKafkaBrokerName) && strings.Contains(f, test.Namespace) {
					cmBytes, _ := json.MarshalIndent(cm, "", " ")
					t.Fatalf("ConfigMap still contains the finalizer %s\n%s\n", f, string(cmBytes))
				}
			}

			return nil
		})

		return broker
	})
}

func waitForBrokerDeletion(ctx context.Context, client *test.Context, t *testing.T) wait.ConditionFunc {
	return func() (bool, error) {
		br, err := client.
			Clients.
			Eventing.
			EventingV1().
			Brokers(test.Namespace).
			Get(ctx, nativeKafkaBrokerName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to get broker %s/%s: %w", test.Namespace, nativeKafkaBrokerName, err)
		}

		brBytes, _ := json.MarshalIndent(br, "", " ")
		t.Logf("Broker still present\n%s\n", string(brBytes))

		return false, nil
	}
}
