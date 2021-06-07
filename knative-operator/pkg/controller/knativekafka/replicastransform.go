package knativekafka

import (
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	KafkaHAComponents = []string{"kafka-ch-controller", "kafka-webhook", "kafka-controller-manager"}
)

func checkHAComponent(name string) bool {
	for _, component := range KafkaHAComponents {
		if name == component {
			return true
		}
	}
	return false
}

// replicasTransform makes kafka-ch-dispatcher keep its current replica count set by controller
// based on vendor/knative.dev/operator/pkg/reconciler/knativeeventing/common/replicasenvvarstransform.go
func replicasTransform(client mf.Client, ha *int32) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && checkHAComponent(u.GetName()) {
			_, err := client.Get(u)
			if errors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			apply := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, apply, nil); err != nil {
				return err
			}
			apply.Spec.Replicas = ha
			if err := scheme.Scheme.Convert(apply, u, nil); err != nil {
				return err
			}
			// The zero-value timestamp defaulted by the conversion causes
			// superfluous updates
			u.SetCreationTimestamp(metav1.Time{})
		}
		if u.GetKind() == "Deployment" && u.GetName() == "kafka-ch-dispatcher" {
			currentU, err := client.Get(u)
			if errors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			apply := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, apply, nil); err != nil {
				return err
			}

			current := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(currentU, current, nil); err != nil {
				return err
			}

			// Keep the existing number of replicas in the cluster for the deployment
			apply.Spec.Replicas = current.Spec.Replicas

			if err := scheme.Scheme.Convert(apply, u, nil); err != nil {
				return err
			}
			// The zero-value timestamp defaulted by the conversion causes
			// superfluous updates
			u.SetCreationTimestamp(metav1.Time{})
		}
		return nil
	}
}
