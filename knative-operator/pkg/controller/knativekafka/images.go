package knativekafka

import (
	"fmt"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

var delimiter = "/"

// ImageTransform updates image with a new registry and tag
func ImageTransform(overrideMap map[string]string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		var podSpec *corev1.PodSpec
		var obj metav1.Object

		switch u.GetKind() {
		case "Deployment":
			deployment := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to Deployment: %w", err)
			}

			obj = deployment
			podSpec = &deployment.Spec.Template.Spec
		case "DaemonSet":
			ds := &appsv1.DaemonSet{}
			if err := scheme.Scheme.Convert(u, ds, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to DaemonSet: %w", err)
			}

			obj = ds
			podSpec = &ds.Spec.Template.Spec
		case "StatefulSet":
			ss := &appsv1.StatefulSet{}
			if err := scheme.Scheme.Convert(u, ss, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to StatefulSet: %w", err)
			}

			obj = ss
			podSpec = &ss.Spec.Template.Spec
		case "Job":
			job := &batchv1.Job{}
			if err := scheme.Scheme.Convert(u, job, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to Job: %w", err)
			}

			obj = job
			podSpec = &job.Spec.Template.Spec
		default:
			// No matches, exit early
			return nil
		}

		containers := podSpec.Containers
		for i := range containers {
			container := &containers[i]

			// Replace direct image YAML references.
			if image, ok := overrideMap[obj.GetName()+delimiter+container.Name]; ok {
				container.Image = image
			} else if image, ok := overrideMap[obj.GetGenerateName()+delimiter+container.Name]; ok {
				container.Image = image
			} else if image, ok := overrideMap[container.Name]; ok {
				container.Image = image
			}

			// Replace env-var based references.
			for j := range container.Env {
				env := &container.Env[j]
				if image, ok := overrideMap[env.Name]; ok {
					env.Value = image
				}
			}
		}

		if err := scheme.Scheme.Convert(obj, u, nil); err != nil {
			return err
		}
		// The zero-value timestamp defaulted by the conversion causes
		// superfluous updates
		u.SetCreationTimestamp(metav1.Time{})
		return nil
	}
}
