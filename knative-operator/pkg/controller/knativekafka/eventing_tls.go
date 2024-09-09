package knativekafka

import (
	"context"
	"fmt"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/eventing/pkg/apis/feature"
	"sigs.k8s.io/controller-runtime/pkg/client"

	serverlessoperatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
)

var (
	tlsResourcesPred = byGroup("cert-manager.io")
)

func (r *ReconcileKnativeKafka) handleTLSResources(ctx context.Context) func(manifests *mf.Manifest, instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	return func(manifests *mf.Manifest, instance *serverlessoperatorv1alpha1.KnativeKafka) error {

		enabled, err := r.isTLSEnabled(ctx, instance)
		if err != nil {
			return err
		}
		if enabled {
			log.Info("Eventing TLS is enabled")
			return nil
		}

		// Delete TLS resources (if present)
		toBeDeleted := manifests.Filter(tlsResourcesPred)
		if err := toBeDeleted.Delete(mf.IgnoreNotFound(true)); err != nil && !isNoMatchError(err) {
			return fmt.Errorf("failed to delete TLS resources: %w", err)
		}

		// Filter out TLS resources from the final list of manifests
		*manifests = manifests.Filter(mf.Not(tlsResourcesPred))

		return nil
	}
}

func (r *ReconcileKnativeKafka) isTLSEnabled(ctx context.Context, instance *serverlessoperatorv1alpha1.KnativeKafka) (bool, error) {
	cm := &corev1.ConfigMap{}
	key := client.ObjectKey{Namespace: instance.GetNamespace(), Name: "config-features"}
	if err := r.client.Get(ctx, key, cm); err != nil {
		return false, fmt.Errorf("failed to get ConfigMap %s: %w", key.String(), err)
	}

	te, ok := cm.Data[feature.TransportEncryption]
	if !ok {
		return false, nil
	}

	f, err := feature.NewFlagsConfigFromMap(map[string]string{
		feature.TransportEncryption: te,
	})
	if err != nil {
		return false, fmt.Errorf("failed to build feature flags from ConfigMap %s: %w", key.String(), err)
	}

	return f.IsPermissiveTransportEncryption() || f.IsStrictTransportEncryption(), nil
}

func byGroup(group string) mf.Predicate {
	return func(u *unstructured.Unstructured) bool {
		return u.GroupVersionKind().Group == group
	}
}
