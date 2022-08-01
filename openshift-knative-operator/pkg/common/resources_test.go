package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"knative.dev/operator/pkg/apis/operator/base"
)

func TestEnsureContainerMemoryLimit(t *testing.T) {
	cases := []struct {
		name     string
		in       []base.ResourceRequirementsOverride
		expected []base.ResourceRequirementsOverride
	}{{
		name: "all nil",
		expected: []base.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("1024Mi"),
				},
			},
		}},
	}, {
		name: "don't override",
		in: []base.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("2048Mi"),
				},
			},
		}},
		expected: []base.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("2048Mi"),
				},
			},
		}},
	}, {
		name: "leave other values alone",
		in: []base.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
			},
		}},
		expected: []base.ResourceRequirementsOverride{{
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
		in: []base.ResourceRequirementsOverride{{
			Container: "foo",
			ResourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
			},
		}},
		expected: []base.ResourceRequirementsOverride{{
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
			s := &base.CommonSpec{DeprecatedResources: c.in}
			EnsureContainerMemoryLimit(s, "foo", resource.MustParse("1024Mi"))

			if !cmp.Equal(s.DeprecatedResources, c.expected) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", s.DeprecatedResources, c.expected, cmp.Diff(s.DeprecatedResources, c.expected))
			}
		})
	}
}
