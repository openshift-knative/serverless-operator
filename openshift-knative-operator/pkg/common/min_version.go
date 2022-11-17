package common

import (
	"fmt"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	// KubernetesMinVersion defines the value for KUBERNETES_MIN_VERSION.
	// Since CSV has the similar validation, we avoid the validation check
	// by setting it to v1.0.0.
	KubernetesMinVersion = "v1.0.0"
)

// InjectCommonEnvironment injects the specified environment variables into the
// deployment and statefulset.
func InjectCommonEnvironment(envs ...corev1.EnvVar) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		var podSpec *corev1.PodSpec
		var convert func(spec *corev1.PodSpec) error

		if u.GetKind() == "StatefulSet" {
			ss := &appsv1.StatefulSet{}
			if err := scheme.Scheme.Convert(u, ss, nil); err != nil {
				return fmt.Errorf("failed to transform Unstructred into StatefulSet: %w", err)
			}
			podSpec = &ss.Spec.Template.Spec
			convert = func(spec *corev1.PodSpec) error {
				ss.Spec.Template.Spec = *podSpec
				return scheme.Scheme.Convert(ss, u, nil)
			}
		}

		if u.GetKind() == "Deployment" {
			var dep = &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, dep, nil); err != nil {
				return fmt.Errorf("failed to transform Unstructred into Deployment: %w", err)
			}
			podSpec = &dep.Spec.Template.Spec
			convert = func(spec *corev1.PodSpec) error {
				dep.Spec.Template.Spec = *podSpec
				return scheme.Scheme.Convert(dep, u, nil)
			}
		}

		if podSpec == nil {
			// Do not need to inject the env valiable.
			return nil
		}

		for i := range podSpec.Containers {
			podSpec.Containers[i].Env = append(podSpec.Containers[i].Env, corev1.EnvVar{
				Name:  "KUBERNETES_MIN_VERSION",
				Value: KubernetesMinVersion,
			})
		}
		return convert(podSpec)
	}
}
