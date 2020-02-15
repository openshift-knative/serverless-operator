package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const knativeServing = "knative-serving"

func TestKnativeServingCleanUp(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)

	defer test.CleanupAll(caCtx)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(caCtx) })

	t.Run("remove knativeserving cr", func(t *testing.T) {
		if err := v1a1test.DeleteKnativeServing(caCtx, knativeServing, knativeServing); err != nil {
			t.Fatal("Failed to remove Knative Serving", err)
		}

		ns, err := caCtx.Clients.Kube.CoreV1().Namespaces().Get(knativeServing+"-ingress", metav1.GetOptions{})
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
		caCtx.Cleanup()
		err := test.WaitForOperatorDepsDeleted(caCtx)
		if err != nil {
			t.Fatalf("Operators still running: %v", err)
		}
	})
}
