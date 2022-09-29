package common

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"knative.dev/operator/pkg/apis/operator/base"
)

// EnsureContainerMemoryLimit makes sure the memory limit for the given container is set
// to the specified amount if not otherwise configured already.
func EnsureContainerMemoryLimit(s *base.CommonSpec, containerName string, memory resource.Quantity) {
	for i, v := range s.DeprecatedResources {
		if v.Container == containerName {
			if v.Limits == nil {
				v.Limits = corev1.ResourceList{}
			}
			if _, ok := v.Limits[corev1.ResourceMemory]; ok {
				return
			}
			v.Limits[corev1.ResourceMemory] = memory
			s.DeprecatedResources[i] = v
			return
		}
	}
	s.DeprecatedResources = append(s.DeprecatedResources, base.ResourceRequirementsOverride{
		Container: containerName,
		ResourceRequirements: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: memory,
			},
		},
	})
}
