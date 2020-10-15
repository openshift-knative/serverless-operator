package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestEnsureContainerMemoryLimit(t *testing.T) {
	cases := []struct {
		name     string
		in       []v1alpha1.ResourceRequirementsOverride
		expected []v1alpha1.ResourceRequirementsOverride
	}{{
		name: "all nil",
		expected: []v1alpha1.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("1024Mi"),
				},
			},
		}},
	}, {
		name: "don't override",
		in: []v1alpha1.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("2048Mi"),
				},
			},
		}},
		expected: []v1alpha1.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("2048Mi"),
				},
			},
		}},
	}, {
		name: "leave other values alone",
		in: []v1alpha1.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
			},
		}},
		expected: []v1alpha1.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("1024Mi"),
				},
			},
		}},
	}, {
		name: "leave request values alone",
		in: []v1alpha1.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
			},
		}},
		expected: []v1alpha1.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("1024Mi"),
				},
			},
		}},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &v1alpha1.CommonSpec{Resources: c.in}
			EnsureContainerMemoryLimit(s, "foo", resource.MustParse("1024Mi"))

			if !cmp.Equal(s.Resources, c.expected) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", s.Resources, c.expected, cmp.Diff(s.Resources, c.expected))
			}
		})
	}
}
