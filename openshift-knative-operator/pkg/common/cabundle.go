package common

import (
	"fmt"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

func ApplyCABundlesTransform() mf.Transformer {
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
		case "StatefulSet":
			ss := &appsv1.StatefulSet{}
			if err := scheme.Scheme.Convert(u, ss, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to StatefulSet: %w", err)
			}

			obj = ss
			podSpec = &ss.Spec.Template.Spec
		default:
			// No matches, exit early
			return nil
		}

		// Let's add the trusted and service CA bundle ConfigMaps as a volume in
		// the PodSpec which will later be mounted to add certs in the pod.
		podSpec.Volumes = AddCABundleConfigMapsToVolumes(podSpec.Volumes)

		// Now that the injected certificates have been added as a volume, let's
		// mount them via volumeMounts in the containers
		for i := range podSpec.Containers {
			c := podSpec.Containers[i] // Create a copy of the container
			AddCABundlesToContainerVolumes(&c)
			podSpec.Containers[i] = c
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
