package e2ekafka

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/e2e"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
)

const (
	eventingName                 = "knative-eventing"
	eventingNamespace            = "knative-eventing"
	knativeKafkaName             = "knative-kafka"
	knativeKafkaChannelNamespace = "knative-eventing"
	knativeKafkaSourceNamespace  = "knative-sources"
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
		if _, err := test.WithOperatorReady(caCtx, "serverless-operator-subscription"); err != nil {
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

	t.Run("verify correct deployment shape for KafkaChannel", func(t *testing.T) {
		for i := range knativeKafkaChannelControlPlaneDeploymentNames {
			deploymentName := knativeKafkaChannelControlPlaneDeploymentNames[i]
			if _, err := test.WithDeploymentReady(caCtx, deploymentName, knativeKafkaChannelNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}
	})

	t.Run("verify correct deployment shape for KafkaSource", func(t *testing.T) {
		for i := range knativeKafkaSourceControlPlaneDeploymentNames {
			deploymentName := knativeKafkaSourceControlPlaneDeploymentNames[i]
			if _, err := test.WithDeploymentReady(caCtx, deploymentName, knativeKafkaSourceNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deploymentName, err)
			}
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		e2e.VerifyNoDisallowedImageReference(t, caCtx, knativeKafkaChannelNamespace)
		e2e.VerifyNoDisallowedImageReference(t, caCtx, knativeKafkaSourceNamespace)
	})

	t.Run("remove knativekafka cr", func(t *testing.T) {
		if err := v1a1test.DeleteKnativeKafka(caCtx, knativeKafkaName, eventingNamespace); err != nil {
			t.Fatal("Failed to remove Knative Kafka", err)
		}

		for i := range knativeKafkaChannelControlPlaneDeploymentNames {
			deploymentName := knativeKafkaChannelControlPlaneDeploymentNames[i]
			if err := test.WithDeploymentGone(caCtx, deploymentName, knativeKafkaChannelNamespace); err != nil {
				t.Fatalf("Deployment %s is not gone: %v", deploymentName, err)
			}
		}

		for i := range knativeKafkaSourceControlPlaneDeploymentNames {
			deploymentName := knativeKafkaSourceControlPlaneDeploymentNames[i]
			if err := test.WithDeploymentGone(caCtx, deploymentName, knativeKafkaSourceNamespace); err != nil {
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
