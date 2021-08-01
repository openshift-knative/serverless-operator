package knativekafka

import (
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

const configMapHashAnnotationKey = "kafka.eventing.knative.dev/configmap-hash"

// configMapHashTransform makes kafka-ch-dispatcher keep its current configmap hash annotation set by controller
func configMapHashTransform(client mf.Client) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
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

			// if no annotations available on the current deployment, do nothing
			if current.Spec.Template.ObjectMeta.Annotations == nil {
				return nil
			}

			// If, the annotation we're looking for doesn't exist in the current deployment,
			// do nothing.
			// Don't even clear it in the target deployment because during an upgrade, this annotation
			// won't be set on the current but it will be set on the target deployment.
			if _, ok := current.Spec.Template.ObjectMeta.Annotations[configMapHashAnnotationKey]; !ok {
				return nil
			}

			if apply.Spec.Template.ObjectMeta.Annotations == nil {
				apply.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}

			// Keep the existing value for the annotation
			apply.Spec.Template.ObjectMeta.Annotations[configMapHashAnnotationKey] = current.Spec.Template.ObjectMeta.Annotations[configMapHashAnnotationKey]

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
