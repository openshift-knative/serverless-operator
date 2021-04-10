package eventing

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/apis"
	kubefake "knative.dev/pkg/client/injection/kube/client/fake"
)

const requiredNs = "knative-eventing"

var (
	eventingNamespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: requiredNs,
		},
	}
)

func TestReconcile(t *testing.T) {
	os.Setenv("IMAGE_foo", "bar")
	os.Setenv("IMAGE_default", "bar2")
	os.Setenv(requiredNsEnvName, requiredNs)

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
	}, {
		name: "Wrong namespace",
		in: ke(func(ke *v1alpha1.KnativeEventing) {
			ke.Namespace = "foo"
		}),
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			ke.Namespace = "foo"
			ke.Status.MarkInstallFailed(`Knative Eventing must be installed into the namespace "knative-eventing"`)
		}),
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Default the namespace to the correct one if not set for brevity.
			if c.in.Namespace == "" {
				c.in.Namespace = requiredNs
			}

			ks := c.in.DeepCopy()
			ctx, _ := kubefake.With(context.Background(), &eventingNamespace)
			ext := NewExtension(ctx)
			ext.Reconcile(context.Background(), ks)

			// Ignore time differences.
			opt := cmp.Comparer(func(apis.VolatileTime, apis.VolatileTime) bool {
				return true
			})

			if !cmp.Equal(ks, c.expected, opt) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", ks, c.expected, cmp.Diff(ks, c.expected, opt))
			}
		})
	}
}

func ke(mods ...func(*v1alpha1.KnativeEventing)) *v1alpha1.KnativeEventing {
	base := &v1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: requiredNs,
		},
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
