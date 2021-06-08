package e2ekafka

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/e2e"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing-kafka/pkg/apis/messaging/v1beta1"
)

const (
	eventingName          = "knative-eventing"
	eventingNamespace     = "knative-eventing"
	knativeKafkaName      = "knative-kafka"
	knativeKafkaNamespace = "knative-eventing"
)

var knativeKafkaChannelControlPlaneDeploymentNames = []string{
	"kafka-ch-controller",
	"kafka-webhook",
}

var knativeKafkaSourceControlPlaneDeploymentNames = []string{
	"kafka-controller-manager",
}

func TestKnativeKafka(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	t.Run("create subscription and wait for CSV to succeed", func(t *testing.T) {
		if _, err := test.WithOperatorReady(caCtx, test.Flags.Subscription); err != nil {
			t.Fatal("Failed", err)
		}
	})

	t.Run("deploy knativeeventing cr and wait for it to be ready", func(t *testing.T) {
		if _, err := v1a1test.WithKnativeEventingReady(caCtx, eventingName, eventingNamespace); err != nil {
			t.Fatal("Failed to deploy KnativeEventing", err)
		}
	})

	t.Run("deploy knativekafka cr and wait for it to be ready", func(t *testing.T) {
		if _, err := v1a1test.WithKnativeKafkaReady(caCtx, knativeKafkaName, eventingNamespace); err != nil {
			t.Fatal("Failed to deploy KnativeKafka", err)
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

	if _, err := caCtx.Clients.Kafka.MessagingV1beta1().KafkaChannels(knativeKafkaNamespace).Create(context.Background(), ch, metav1.CreateOptions{}); err != nil {
		t.Fatal("Failed to create channel to trigger the dispatcher deployment", err)
	}

	t.Cleanup(func() {
		_ = caCtx.Clients.Kafka.MessagingV1beta1().KafkaChannels(knativeKafkaNamespace).Delete(context.Background(), "", metav1.DeleteOptions{})
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

	t.Run("verify Kafka control plane metrics work correctly", func(t *testing.T) {
		// Kafka control plane metrics should work
		if err := monitoringe2e.VerifyMetrics(caCtx, monitoringe2e.KafkaQueries); err != nil {
			t.Fatal("Failed to verify that Kafka control plane metrics work correctly", err)
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		e2e.VerifyNoDisallowedImageReference(t, caCtx, knativeKafkaNamespace)
	})

	t.Run("remove knativekafka cr", func(t *testing.T) {
		if err := v1a1test.DeleteKnativeKafka(caCtx, knativeKafkaName, eventingNamespace); err != nil {
			t.Fatal("Failed to remove Knative Kafka", err)
		}

		for i := range knativeKafkaChannelControlPlaneDeploymentNames {
			deploymentName := knativeKafkaChannelControlPlaneDeploymentNames[i]
			if err := test.WithDeploymentGone(caCtx, deploymentName, knativeKafkaNamespace); err != nil {
				t.Fatalf("Deployment %s is not gone: %v", deploymentName, err)
			}
		}

		for i := range knativeKafkaSourceControlPlaneDeploymentNames {
			deploymentName := knativeKafkaSourceControlPlaneDeploymentNames[i]
			if err := test.WithDeploymentGone(caCtx, deploymentName, knativeKafkaNamespace); err != nil {
				t.Fatalf("Deployment %s is not gone: %v", deploymentName, err)
			}
		}
	})

	t.Run("remove knativeeventing cr", func(t *testing.T) {
		if err := v1a1test.DeleteKnativeEventing(caCtx, eventingName, eventingNamespace); err != nil {
			t.Fatal("Failed to remove Knative Eventing", err)
		}
	})

	t.Run("undeploy serverless operator and check dependent operators removed", func(t *testing.T) {
		caCtx.Cleanup(t)
		if err := test.WaitForOperatorDepsDeleted(caCtx); err != nil {
			t.Fatalf("Operators still running: %v", err)
		}
	})
}
