package common

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

// EnsureContainerMemoryLimit makes sure the memory limit for the given container is set
// to the specified amount if not otherwise configured already.
func EnsureContainerMemoryLimit(s *v1alpha1.CommonSpec, containerName string, memory resource.Quantity) {
	for i, v := range s.Resources {
		if v.Container == containerName {
			if v.Limits == nil {
				v.Limits = corev1.ResourceList{}
			}
			if _, ok := v.Limits[corev1.ResourceMemory]; ok {
				return
			}
			v.Limits[corev1.ResourceMemory] = memory
			s.Resources[i] = v
			return
		}
	}
	s.Resources = append(s.Resources, v1alpha1.ResourceRequirementsOverride{
		Container: containerName,
		ResourceRequirements: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: memory,
			},
		},
	})
}
