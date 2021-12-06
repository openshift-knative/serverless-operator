package e2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	servingName      = "knative-serving"
	servingNamespace = "knative-serving"
	haReplicas       = 2
)

func TestKnativeServing(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	t.Run("create subscription and wait for CSV to succeed", func(t *testing.T) {
		if _, err := test.WithOperatorReady(caCtx, test.Flags.Subscription); err != nil {
			t.Fatal("Failed", err)
		}
	})

	t.Run("deploy knativeserving cr and wait for it to be ready", func(t *testing.T) {
		if _, err := v1a1test.WithKnativeServingReady(caCtx, servingName, servingNamespace); err != nil {
			t.Fatal("Failed to deploy KnativeServing", err)
		}
	})

	t.Run("verify health metrics work correctly", func(t *testing.T) {
		// Serving should be up
		if err := monitoringe2e.VerifyHealthStatusMetric(caCtx, "serving_status", "1"); err != nil {
			t.Fatal("Failed to verify that health metrics work correctly for Serving", err)
		}
	})

	t.Run("verify correct deployment shape", func(t *testing.T) {
		// Check the desired scale of deployments in the knative serving namespace
		for _, deployment := range []string{"activator", "controller", "autoscaler-hpa"} {
			if err := test.CheckDeploymentScale(caCtx, servingNamespace, deployment, haReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings for %q: %v", deployment, err)
			}
		}
		// Check the status of deployments in the knative serving namespace
		for _, deployment := range []string{"activator", "autoscaler", "autoscaler-hpa", "controller", "webhook"} {
			if _, err := test.WithDeploymentReady(caCtx, deployment, servingNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment, err)
			}
		}
		// Check the desired scale of deployments in the ingress namespace.
		ingressDeployments := []string{"net-kourier-controller", "3scale-kourier-gateway"}
		for _, deployment := range ingressDeployments {
			if err := test.CheckDeploymentScale(caCtx, servingNamespace+"-ingress", deployment, haReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings: %v", err)
			}
		}
		// Check the status of deployments in the ingress namespace.
		for _, deployment := range ingressDeployments {
			if _, err := test.WithDeploymentReady(caCtx, deployment, servingNamespace+"-ingress"); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment, err)
			}
		}
	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		VerifyNoDisallowedImageReference(t, caCtx, servingNamespace)
	})

	t.Run("remove knativeserving cr", func(t *testing.T) {
		if err := v1a1test.DeleteKnativeServing(caCtx, servingName, servingNamespace); err != nil {
			t.Fatal("Failed to remove Knative Serving", err)
		}

		ns, err := caCtx.Clients.Kube.CoreV1().Namespaces().Get(context.Background(), servingNamespace+"-ingress", metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			// Namespace is already gone, all good!
			return
		} else if err != nil {
			t.Fatal("Failed fetching ingress namespace", err)
		}

		// If the namespace is not gone yet, check if it's terminating.
		if ns.Status.Phase != corev1.NamespaceTerminating {
			t.Fatalf("Ingress namespace phase = %v, want %v", ns.Status.Phase, corev1.NamespaceTerminating)
		}
	})

	t.Run("undeploy serverless operator and check dependent operators removed", func(t *testing.T) {
		caCtx.Cleanup(t)
		if err := test.WaitForOperatorDepsDeleted(caCtx); err != nil {
			t.Fatalf("Operators still running: %v", err)
		}
	})
}
