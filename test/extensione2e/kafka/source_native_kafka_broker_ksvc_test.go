package knativekafkae2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"

	corev1 "k8s.io/api/core/v1"

	"github.com/openshift-knative/serverless-operator/test/eventinge2e"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/eventing-kafka-broker/control-plane/pkg/kafka"
	kafkatestpkg "knative.dev/eventing-kafka-broker/test/pkg"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
)

const (
	nativeKafkaBrokerName = "smoke-test-native-kafka-broker"
)

func TestSourceToNativeKafkaBrokerToKnativeService(t *testing.T) {
	eventinge2e.KnativeSourceBrokerTriggerKnativeService(t, createBrokerFunc(t, kafka.BrokerClass), func(ctx *test.Context) {
		t.Run("verify Kafka Broker data plane metrics work correctly", func(t *testing.T) {
			if err := monitoringe2e.VerifyMetrics(ctx, monitoringe2e.KafkaBrokerDataPlaneQueries); err != nil {
				t.Fatal("Failed to verify that Kafka Broker data plane metrics work correctly", err)
			}
		})
	})
}

func TestSourceToNamespacedKafkaBrokerToKnativeService(t *testing.T) {
	eventinge2e.KnativeSourceBrokerTriggerKnativeService(t, createBrokerFunc(t, kafka.NamespacedBrokerClass), func(ctx *test.Context) {
		t.Run("verify namespaced Kafka Broker data plane metrics work correctly", func(t *testing.T) {
			if err := monitoringe2e.VerifyMetrics(ctx, monitoringe2e.NamespacedKafkaBrokerDataPlaneQueries(test.Namespace)); err != nil {
				t.Fatal("Failed to verify that namespaced Kafka Broker data plane metrics work correctly", err)
			}
		})
	})
}

func createBrokerFunc(t *testing.T, brokerClass string) func(client *test.Context) *eventingv1.Broker {
	return func(client *test.Context) *eventingv1.Broker {
		kafkaBrokerConfigName := "kafka-broker-config"
		kafkaBrokerConfigNamespace := "knative-eventing"
		var brokerConfigMap *corev1.ConfigMap
		if brokerClass == kafka.NamespacedBrokerClass {
			kafkaBrokerConfigNamespace = test.Namespace
			brokerConfigMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kafkaBrokerConfigName,
					Namespace: kafkaBrokerConfigNamespace,
				},
				Data: map[string]string{
					kafka.BootstrapServersConfigMapKey:              kafkatestpkg.BootstrapServersPlaintext,
					kafka.DefaultTopicNumPartitionConfigMapKey:      fmt.Sprintf("%d", kafkatestpkg.NumPartitions),
					kafka.DefaultTopicReplicationFactorConfigMapKey: fmt.Sprintf("%d", kafkatestpkg.ReplicationFactor),
				},
			}
		}
		nativeKafkaBroker := &eventingv1.Broker{
			ObjectMeta: metav1.ObjectMeta{
				Name:        nativeKafkaBrokerName,
				Namespace:   test.Namespace,
				Annotations: map[string]string{"eventing.knative.dev/broker.class": brokerClass},
			},
			Spec: eventingv1.BrokerSpec{
				Config: &duckv1.KReference{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       kafkaBrokerConfigName,
					Namespace:  kafkaBrokerConfigNamespace,
				},
			},
		}

		// Create Kafka Broker ConfigMap for Namespaced broker only
		if brokerClass == kafka.NamespacedBrokerClass {
			_, err := client.Clients.Kube.CoreV1().ConfigMaps(test.Namespace).Create(context.Background(), brokerConfigMap, metav1.CreateOptions{})
			if err != nil {
				t.Fatal("Unable to create KafkaBroker ConfigMap: ", err)
			}
		}

		// Create the (native) Kafka Broker
		broker, err := client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Create(context.Background(), nativeKafkaBroker, metav1.CreateOptions{})
		if err != nil {
			t.Fatal("Unable to create Kafka Backed Broker: ", err)
		}

		client.AddToCleanup(func() error {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
			defer cancel()

			pp := metav1.DeletePropagationForeground
			err := client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Delete(context.Background(), nativeKafkaBrokerName, metav1.DeleteOptions{
				PropagationPolicy: &pp,
			})
			if err != nil {
				t.Fatal(err)
			}

			err = wait.PollImmediateUntil(2*time.Second, waitForBrokerDeletion(ctx, client, t), ctx.Done())
			if err != nil {
				t.Fatal(err)
			}

			if brokerClass == kafka.NamespacedBrokerClass {
				if err := client.Clients.Kube.CoreV1().ConfigMaps(test.Namespace).Delete(ctx, kafkaBrokerConfigName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
					t.Errorf("Failed to delete ConfigMap %s/%s: %v", test.Namespace, kafkaBrokerConfigName, err)
				}
			} else {
				cm, err := client.Clients.Kube.
					CoreV1().
					ConfigMaps(nativeKafkaBroker.Spec.Config.Namespace).
					Get(ctx, nativeKafkaBroker.Spec.Config.Name, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("Failed to get ConfigMap")
				}
				for _, f := range cm.GetFinalizers() {
					if strings.Contains(f, nativeKafkaBrokerName) && strings.Contains(f, test.Namespace) {
						cmBytes, _ := json.MarshalIndent(cm, "", " ")
						t.Fatalf("ConfigMap still contains the finalizer %s\n%s\n", f, string(cmBytes))
					}
				}
			}

			return nil
		})

		return broker
	}
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
