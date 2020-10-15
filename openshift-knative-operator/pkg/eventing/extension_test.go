package eventing

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestReconcile(t *testing.T) {
	os.Setenv("IMAGE_foo", "bar")
	os.Setenv("IMAGE_default", "bar2")

	cases := []struct {
		name     string
		in       *v1alpha1.KnativeEventing
		expected *v1alpha1.KnativeEventing
	}{{
		name:     "all nil",
		in:       &v1alpha1.KnativeEventing{},
		expected: ke(),
	}, {
		name: "With inclusion sinkbinding setting",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				SinkBindingSelectionMode: "inclusion",
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			ke.Spec.SinkBindingSelectionMode = "inclusion"
		}),
	}, {
		name: "With exclusion sinkbinding setting",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				SinkBindingSelectionMode: "exclusion",
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			ke.Spec.SinkBindingSelectionMode = "exclusion"
		}),
	}, {
		name: "With empty sinkbinding setting",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				SinkBindingSelectionMode: "",
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			ke.Spec.SinkBindingSelectionMode = "inclusion"
		}),
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ks := c.in.DeepCopy()
			ext := NewExtension(context.Background())
			ext.Reconcile(context.Background(), ks)

			if !cmp.Equal(ks, c.expected) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", ks, c.expected, cmp.Diff(ks, c.expected))
			}
		})
	}
}

func ke(mods ...func(*v1alpha1.KnativeEventing)) *v1alpha1.KnativeEventing {
	base := &v1alpha1.KnativeEventing{
		Spec: v1alpha1.KnativeEventingSpec{
			SinkBindingSelectionMode: "inclusion",
			CommonSpec: v1alpha1.CommonSpec{
				Registry: v1alpha1.Registry{
					Default: "bar2",
					Override: map[string]string{
						"default": "bar2",
						"foo":     "bar",
					},
				},
				Resources: []v1alpha1.ResourceRequirementsOverride{{
					Container: "eventing-webhook",
					ResourceRequirements: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("1024Mi"),
						},
					},
				}},
			},
		},
	}

	for _, mod := range mods {
		mod(base)
	}

	return base
}
