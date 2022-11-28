package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/openshift-knative/serverless-operator/test"
)

func VerifyNamespaceAnnotations(t *testing.T, caCtx *test.Context, namespace string) {
	ctx := context.Background()
	annotation := "custom-test-annotation"
	label := "custom-test-label"
	value := uuid.New().String()

	t.Run("Namespace "+namespace+" additional labels and annotations", func(t *testing.T) {
		t.Run("Add "+label+" label", func(t *testing.T) {
			if err := ensureMetadataOnNamespace(caCtx, ctx, namespace, label, value, labelKVMetadata); err != nil {
				t.Fatal(err)
			}
			if err := verifyMetadataOnNamespace(caCtx, ctx, namespace, label, value, labelKVMetadata); err != nil {
				t.Fatal(err)
			}
		})

		t.Run("Add "+annotation+" annotations", func(t *testing.T) {
			if err := ensureMetadataOnNamespace(caCtx, ctx, namespace, annotation, value, annotationsKVMetadata); err != nil {
				t.Fatal(err)
			}
			if err := verifyMetadataOnNamespace(caCtx, ctx, namespace, annotation, value, annotationsKVMetadata); err != nil {
				t.Fatal(err)
			}
		})
	})
}

func ensureMetadataOnNamespace(caCtx *test.Context, ctx context.Context, namespace string, key string, value string, getKVMetadata getKVMetadata) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ns, err := caCtx.Clients.Kube.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			return err
		}

		metadata := getKVMetadata(ns)
		metadata[key] = value

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

func verifyMetadataOnNamespace(caCtx *test.Context, ctx context.Context, namespace string, key string, value string, getKVMetadata getKVMetadata) error {
	time.Sleep(10 * time.Second) // "Eventually" is still present

	ns, err := caCtx.Clients.Kube.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	metadata := getKVMetadata(ns)

	v, ok := metadata[key]
	if !ok {
		return fmt.Errorf("no label %s with value %s on namespace %s", key, value, namespace)
	}
	if v != value {
		return fmt.Errorf("expected label %s value %s, got %s", key, value, v)
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
