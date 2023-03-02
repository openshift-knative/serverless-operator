package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/openshift-knative/serverless-operator/test"
)

func VerifyNamespaceMetadata(t *testing.T, caCtx *test.Context, namespace string) {
	ctx := context.Background()

	labels := map[string]string{
		"app.kubernetes.io/instance":    "openshift-serverless",
		"argocd.argoproj.io/managed-by": "openshift-gitops",
	}

	annotations := map[string]string{
		"argocd.argoproj.io/sync-wave": "4",
	}

	t.Run("Namespace "+namespace+" additional labels and annotations", func(t *testing.T) {
		t.Run("Add labels", func(t *testing.T) {
			if err := ensureMetadataOnNamespace(ctx, caCtx, namespace, labels, labelKVMetadata); err != nil {
				t.Fatal(err)
			}
			if err := verifyMetadataOnNamespace(ctx, caCtx, namespace, labels, labelKVMetadata); err != nil {
				t.Fatal(err)
			}
		})

		t.Run("Add annotations", func(t *testing.T) {
			if err := ensureMetadataOnNamespace(ctx, caCtx, namespace, annotations, annotationsKVMetadata); err != nil {
				t.Fatal(err)
			}
			if err := verifyMetadataOnNamespace(ctx, caCtx, namespace, annotations, annotationsKVMetadata); err != nil {
				t.Fatal(err)
			}
		})
	})
}

func ensureMetadataOnNamespace(ctx context.Context, caCtx *test.Context, namespace string, additional map[string]string, getKVMetadata getKVMetadata) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ns, err := caCtx.Clients.Kube.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			return err
		}

		metadata := getKVMetadata(ns)
		for k, v := range additional {
			metadata[k] = v
		}

		if _, err := caCtx.Clients.Kube.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func verifyMetadataOnNamespace(ctx context.Context, caCtx *test.Context, namespace string, expected map[string]string, getKVMetadata getKVMetadata) error {
	// Operator doesn't reconcile the namespace, so the problem will be triggered on restart or
	// on upgrades, so simulate a SO restart.
	if err := caCtx.DeleteOperatorPods(ctx); err != nil {
		return err
	}

	if err := caCtx.WaitForOperatorPodsReady(ctx); err != nil {
		return fmt.Errorf("failed while waiting for operator pods to become ready: %w", err)
	}

	time.Sleep(10 * time.Second) // "Eventually" is still present

	ns, err := caCtx.Clients.Kube.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	metadata := getKVMetadata(ns)

	for k, v := range expected {
		got, ok := metadata[k]
		if !ok {
			return fmt.Errorf("no metadata %s with value %s on namespace %s, got\n%+v", k, v, namespace, metadata)
		}
		if got != v {
			return fmt.Errorf("expected metadata %s value %s, got %s", k, v, got)
		}
	}

	return nil
}

type getKVMetadata func(ns *corev1.Namespace) map[string]string

func labelKVMetadata(ns *corev1.Namespace) map[string]string {
	if ns.Labels == nil {
		ns.Labels = make(map[string]string, 1)
	}
	return ns.Labels
}

func annotationsKVMetadata(ns *corev1.Namespace) map[string]string {
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string, 1)
	}
	return ns.Annotations
}
