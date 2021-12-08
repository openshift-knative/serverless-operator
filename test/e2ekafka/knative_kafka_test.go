package e2ekafka

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/e2e"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing-kafka/pkg/apis/messaging/v1beta1"
)

const (
	eventingName          = "knative-eventing"
	eventingNamespace     = "knative-eventing"
	knativeKafkaNamespace = "knative-eventing"
)

var knativeKafkaChannelControlPlaneDeploymentNames = []string{
	"kafka-ch-controller",
	"kafka-webhook",
	"kafka-ch-dispatcher",
}

var knativeKafkaSourceControlPlaneDeploymentNames = []string{
	"kafka-controller-manager",
}

var knativeKafkaBrokerControlPlaneDeploymentNames = []string{
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

	t.Run("deploy channel cr and wait for it to be ready", func(t *testing.T) {
		if _, err := caCtx.Clients.Kafka.MessagingV1beta1().KafkaChannels(knativeKafkaNamespace).Create(context.Background(), ch, metav1.CreateOptions{}); err != nil {
			t.Fatal("Failed to create channel to trigger the dispatcher deployment", err)
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
		for i := range knativeKafkaChannelControlPlaneDeploymentNames {
			deploymentName := knativeKafkaChannelControlPlaneDeploymentNames[i]
			if _, err := test.WithDeploymentReady(caCtx, deploymentName, knativeKafkaNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}
	})

	t.Run("verify correct deployment shape for KafkaSource", func(t *testing.T) {
		for i := range knativeKafkaSourceControlPlaneDeploymentNames {
			deploymentName := knativeKafkaSourceControlPlaneDeploymentNames[i]
			if _, err := test.WithDeploymentReady(caCtx, deploymentName, knativeKafkaNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}
	})

	t.Run("verify correct deployment shape for Kafka Broker", func(t *testing.T) {
		for i := range knativeKafkaBrokerControlPlaneDeploymentNames {
			deploymentName := knativeKafkaBrokerControlPlaneDeploymentNames[i]
			if _, err := test.WithDeploymentReady(caCtx, deploymentName, knativeKafkaNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}
	})

	t.Run("verify Kafka control plane metrics work correctly", func(t *testing.T) {
		// Kafka control plane metrics should work
		if err := monitoringe2e.VerifyMetrics(caCtx, monitoringe2e.KafkaQueries); err != nil {
			t.Fatal("Failed to verify that Kafka control plane metrics work correctly", err)
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
}
