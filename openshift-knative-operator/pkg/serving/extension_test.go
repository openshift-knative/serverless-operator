package serving

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
	os.Setenv("IMAGE_queue-proxy", "baz")

	cases := []struct {
		name     string
		in       *v1alpha1.KnativeServing
		expected *v1alpha1.KnativeServing
	}{{
		name:     "all nil",
		in:       &v1alpha1.KnativeServing{},
		expected: ks(),
	}, {
		name: "different HA settings",
		in: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				HighAvailability: &v1alpha1.HighAvailability{
					Replicas: 3,
				},
			},
		},
		expected: ks(func(ks *v1alpha1.KnativeServing) {
			ks.Spec.HighAvailability.Replicas = 3
		}),
	}, {
		name: "different certificate settings",
		in: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				ControllerCustomCerts: v1alpha1.CustomCerts{
					Type: "Secret",
					Name: "foo",
				},
			},
		},
		expected: ks(func(ks *v1alpha1.KnativeServing) {
			ks.Spec.ControllerCustomCerts.Type = "Secret"
			ks.Spec.ControllerCustomCerts.Name = "foo"
		}),
	}, {
		name: "override image settings",
		in: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Registry: v1alpha1.Registry{
						Override: map[string]string{
							"foo":         "not",
							"queue-proxy": "correct",
						},
					},
				},
			},
		},
		expected: ks(),
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

func ks(mods ...func(*v1alpha1.KnativeServing)) *v1alpha1.KnativeServing {
	base := &v1alpha1.KnativeServing{
		Spec: v1alpha1.KnativeServingSpec{
			CommonSpec: v1alpha1.CommonSpec{
				Config: v1alpha1.ConfigMapData{
					"deployment": map[string]string{
						"queueSidecarImage": "baz",
					},
					"network": map[string]string{
						"domainTemplate": "{{.Name}}-{{.Namespace}}.{{.Domain}}",
						"ingress.class":  "kourier.ingress.networking.knative.dev",
					},
				},
				Registry: v1alpha1.Registry{
					Default: "bar2",
					Override: map[string]string{
						"default":     "bar2",
						"foo":         "bar",
						"queue-proxy": "baz",
					},
				},
				Resources: []v1alpha1.ResourceRequirementsOverride{{
					Container: "webhook",
					ResourceRequirements: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("1024Mi"),
						},
					},
				}},
			},
			ControllerCustomCerts: v1alpha1.CustomCerts{
				Type: "ConfigMap",
				Name: "config-service-ca",
			},
			HighAvailability: &v1alpha1.HighAvailability{
				Replicas: 2,
			},
		},
	}

	for _, mod := range mods {
		mod(base)
	}

	return base
}
