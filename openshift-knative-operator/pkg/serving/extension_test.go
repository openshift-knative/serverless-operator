package serving

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/apis"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	ocpfake "github.com/openshift-knative/serverless-operator/pkg/client/injection/client/fake"
)

func TestReconcile(t *testing.T) {
	os.Setenv("IMAGE_foo", "bar")
	os.Setenv("IMAGE_default", "bar2")
	os.Setenv("IMAGE_queue-proxy", "baz")

	defaultIngress := &configv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.IngressSpec{
			Domain: "routing.example.com",
		},
	}

	cases := []struct {
		name     string
		in       *v1alpha1.KnativeServing
		objs     []runtime.Object
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
		name: "existing logging route",
		in:   &v1alpha1.KnativeServing{},
		objs: []runtime.Object{
			defaultIngress,
			&routev1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "openshift-logging",
					Name:      "kibana",
				},
				Status: routev1.RouteStatus{
					Ingress: []routev1.RouteIngress{{
						Host: "logging.example.com",
					}},
				},
			},
		},
		expected: ks(func(ks *v1alpha1.KnativeServing) {
			common.Configure(&ks.Spec.CommonSpec, "observability", "logging.revision-url-template",
				fmt.Sprintf(loggingURLTemplate, "logging.example.com"))
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
	}, {
		name: "respects different status",
		in: ks(func(ks *v1alpha1.KnativeServing) {
			ks.Status.MarkDependenciesInstalled()
		}),
		expected: ks(func(ks *v1alpha1.KnativeServing) {
			ks.Status.MarkDependenciesInstalled()
		}),
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			objs := c.objs
			if objs == nil {
				objs = []runtime.Object{defaultIngress}
			}
			ctx, _ := ocpfake.With(context.Background(), objs...)

			ks := c.in.DeepCopy()
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

func ks(mods ...func(*v1alpha1.KnativeServing)) *v1alpha1.KnativeServing {
	base := &v1alpha1.KnativeServing{
		Spec: v1alpha1.KnativeServingSpec{
			CommonSpec: v1alpha1.CommonSpec{
				Config: v1alpha1.ConfigMapData{
					"deployment": map[string]string{
						"queueSidecarImage": "baz",
					},
					"domain": map[string]string{
						"routing.example.com": "",
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

	base.Status.MarkDependencyInstalling("Kourier")

	for _, mod := range mods {
		mod(base)
	}

	return base
}
