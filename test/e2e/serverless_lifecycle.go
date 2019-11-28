package e2e

import (
	"github.com/openshift-knative/serverless-operator/test"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	knativeServing = "knative-serving"
)

func deployServerlessOperator(t *testing.T, ctx *test.Context) {
	t.Run("deploy Serverless operator", func(t *testing.T) {
		_, err := test.WithOperatorReady(ctx, "serverless-operator-subscription")
		if err != nil {
			t.Fatal("Failed", err)
		}
	})
}

func removeServerlessOperator(t *testing.T, ctx *test.Context) {
	if t.Failed() && runsOnOpenshiftCI() {
		t.Log("Skipping removal of Serverless as tests failed and we are running on Openshift CI")
		return
	}
	t.Run("remove Serverless operator", func(t *testing.T) {
		ctx.Cleanup()
		err := test.WaitForOperatorDepsDeleted(ctx)
		if err != nil {
			t.Fatalf("Operators still running: %v", err)
		}
	})
}

func deployKnativeServingResource(t *testing.T, ctx *test.Context) {
	t.Run("deploy KnativeServing resource", func(t *testing.T) {
		_, err := test.WithKnativeServingReady(ctx, knativeServing, knativeServing)
		if err != nil {
			t.Fatal("Failed to deploy KnativeServing resource", err)
		}
	})
}

func removeKnativeServingResource(t *testing.T, ctx *test.Context) {
	if t.Failed() && runsOnOpenshiftCI() {
		t.Log("Skipping removal of KnativeServing resource as tests failed and we are running on Openshift CI")
		return
	}
	t.Run("remove KnativeServing resource", func(t *testing.T) {
		if err := test.DeleteKnativeServing(ctx, knativeServing, knativeServing); err != nil {
			t.Fatal("Failed to remove Knative Serving resource", err)
		}

		ns, err := ctx.Clients.Kube.CoreV1().Namespaces().Get(
			knativeServing + "-ingress", metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			// Namespace is already gone, all good!
			return
		} else if err != nil {
			t.Fatal("Failed fetching ingress namespace", err)
		}

		// If the namespace is not gone yet, check if it's terminating.
		if ns.Status.Phase != corev1.NamespaceTerminating {
			t.Fatalf("Ingress namespace phase = %v, want %v",
				ns.Status.Phase, corev1.NamespaceTerminating)
		}
	})
}

func cleanupOnInterrupt(t *testing.T, contexts ...*test.Context) {
	if runsOnOpenshiftCI() {
		t.Log("Skipping register of CleanUp on interrupt as we are running on Openshift CI")
	}
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(contexts...) })
}

func cleanupAll(t *testing.T, contexts ...*test.Context) {
	if t.Failed() && runsOnOpenshiftCI() {
		t.Log("Skipping cleanup as test have failed and we are running on Openshift CI")
		return
	}
	test.CleanupAll(contexts...)
}

func runsOnOpenshiftCI() bool {
	_, ok := os.LookupEnv("OPENSHIFT_BUILD_NAMESPACE")
	return ok
}
