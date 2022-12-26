package e2ekafka

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing-kafka/pkg/apis/messaging/v1beta1"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/e2e"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	"github.com/openshift-knative/serverless-operator/test/upgrade"
)

const (
	eventingName          = "knative-eventing"
	eventingNamespace     = "knative-eventing"
	knativeKafkaNamespace = "knative-eventing"
	defaultNamespace      = "default"
)

var kafkaChannelDeployments = []string{
	"kafka-channel-dispatcher",
	"kafka-channel-receiver",
}

var kafkaSourceDeployments = []string{
	"kafka-source-dispatcher",
}

var kafkaControlPlaneDeployments = []string{
	"kafka-controller",
	"kafka-webhook-eventing",
}

func TestKnativeKafka(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	// Ensure KnativeEventing is already installed.
	if ev, err := caCtx.Clients.Operator.KnativeEventings(eventingNamespace).
		Get(context.Background(), eventingName, metav1.GetOptions{}); err != nil || !ev.Status.IsReady() {
		t.Fatal("KnativeEventing CR must be ready:", err)
	}

	ch := &v1beta1.KafkaChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testchannel",
			Namespace: knativeKafkaNamespace,
		},
		Spec: v1beta1.KafkaChannelSpec{
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	brokerCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testbrokerconfig",
			Namespace: defaultNamespace,
		},
		Data: map[string]string{
			"bootstrap.servers":                "my-cluster-kafka-bootstrap.kafka:9092",
			"default.topic.partitions":         "10",
			"default.topic.replication.factor": "3",
		},
	}

	broker := &eventingv1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testbroker",
			Namespace: defaultNamespace,
			Annotations: map[string]string{
				"eventing.knative.dev/broker.class": "KafkaNamespaced",
			},
		},
		Spec: eventingv1.BrokerSpec{
			Config: &duckv1.KReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Namespace:  defaultNamespace,
				Name:       "testbrokerconfig",
			},
		},
	}

	t.Run("deploy channel cr and wait for it to be ready", func(t *testing.T) {
		if _, err := caCtx.Clients.Kafka.MessagingV1beta1().KafkaChannels(knativeKafkaNamespace).Create(context.Background(), ch, metav1.CreateOptions{}); err != nil {
			t.Fatal("Failed to create channel to trigger the dispatcher deployment", err)
		}
	})

	t.Run("create config for a namespaced broker and wait for it to be ready", func(t *testing.T) {
		if _, err := caCtx.Clients.Kube.CoreV1().ConfigMaps(defaultNamespace).Create(context.Background(), brokerCm, metav1.CreateOptions{}); err != nil {
			t.Fatal("Failed to create config for the namespaced broker to trigger the dispatcher deployment", err)
		}
	})

	t.Run("deploy a namespaced broker cr and wait for it to be ready", func(t *testing.T) {
		if _, err := caCtx.Clients.Eventing.EventingV1().Brokers(defaultNamespace).Create(context.Background(), broker, metav1.CreateOptions{}); err != nil {
			t.Fatal("Failed to create namespaced broker to trigger the dispatcher deployment", err)
		}
	})

	t.Run("verify health metrics work correctly", func(t *testing.T) {
		// Eventing should be up
		if err := monitoringe2e.VerifyHealthStatusMetric(caCtx, "eventing_status", "1"); err != nil {
			t.Fatal("Failed to verify that health metrics work correctly for Eventing", err)
		}
		// KnativeKafka should be up
		if err := monitoringe2e.VerifyHealthStatusMetric(caCtx, "kafka_status", "1"); err != nil {
			t.Fatal("Failed to verify that health metrics work correctly for KnativeKafka", err)
		}
	})

	t.Run("verify correct deployment shape for KafkaChannel", func(t *testing.T) {
		for i := range kafkaChannelDeployments {
			deploymentName := kafkaChannelDeployments[i]
			if err := test.WithWorkloadReady(caCtx, deploymentName, knativeKafkaNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}
	})

	t.Run("verify correct deployment shape for KafkaSource", func(t *testing.T) {
		for i := range kafkaSourceDeployments {
			deploymentName := kafkaSourceDeployments[i]
			if err := test.WithWorkloadReady(caCtx, deploymentName, knativeKafkaNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}
	})

	t.Run("verify correct deployment shape for Kafka control plane", func(t *testing.T) {
		for i := range kafkaControlPlaneDeployments {
			deploymentName := kafkaControlPlaneDeployments[i]
			if err := test.WithWorkloadReady(caCtx, deploymentName, knativeKafkaNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}
	})

	t.Run("verify Kafka control plane metrics work correctly", func(t *testing.T) {
		if err := monitoringe2e.VerifyMetrics(caCtx, monitoringe2e.KafkaQueries); err != nil {
			t.Fatal("Failed to verify that Kafka control plane metrics work correctly", err)
		}
	})

	t.Run("verify Kafka Broker data plane metrics work correctly", func(t *testing.T) {
		if err := monitoringe2e.VerifyMetrics(caCtx, monitoringe2e.KafkaBrokerDataPlaneQueries); err != nil {
			t.Fatal("Failed to verify that Kafka Broker data plane metrics work correctly", err)
		}
	})

	t.Run("verify Kafka controller metrics work correctly", func(t *testing.T) {
		if err := monitoringe2e.VerifyMetrics(caCtx, monitoringe2e.KafkaControllerQueries); err != nil {
			t.Fatal("Failed to verify that Kafka Broker data plane metrics work correctly", err)
		}
	})

	t.Run("verify namespaced Kafka Broker data plane metrics work correctly", func(t *testing.T) {
		if err := monitoringe2e.VerifyMetrics(caCtx, monitoringe2e.NamespacedKafkaBrokerDataPlaneQueries); err != nil {
			t.Fatal("Failed to verify that namespaced Kafka Broker data plane metrics work correctly", err)
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		e2e.VerifyNoDisallowedImageReference(t, caCtx, knativeKafkaNamespace)
	})

	t.Run("remove channel cr", func(t *testing.T) {
		if err := caCtx.Clients.Kafka.MessagingV1beta1().KafkaChannels(knativeKafkaNamespace).Delete(context.Background(), ch.Name, metav1.DeleteOptions{}); err != nil {
			t.Fatal("Failed to remove Knative Channel", err)
		}
	})

	t.Run("Verify job succeeded", func(t *testing.T) {
		upgrade.VerifyPostInstallJobs(caCtx, upgrade.VerifyPostJobsConfig{
			Namespace:    knativeKafkaNamespace,
			FailOnNoJobs: true,
		})
	})
}
