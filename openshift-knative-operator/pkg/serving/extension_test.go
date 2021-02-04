package serving

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	ocpfake "github.com/openshift-knative/serverless-operator/pkg/client/injection/client/fake"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/injection"
)

var servingNamespace = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "knative-serving",
	},
}

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
		name                    string
		in                      *v1alpha1.KnativeServing
		objs                    []runtime.Object
		expected                *v1alpha1.KnativeServing
		shouldDisableMonitoring bool
	}{{
		name:                    "all nil",
		in:                      &v1alpha1.KnativeServing{},
		expected:                ks(),
		shouldDisableMonitoring: false,
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
		shouldDisableMonitoring: false,
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
		shouldDisableMonitoring: false,
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
			common.Configure(&ks.Spec.CommonSpec, monitoring.ObservabilityCMName, "logging.revision-url-template",
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
		shouldDisableMonitoring: false,
	}, {
		name: "respect already configured metrics backend",
		in: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "prometheus"}},
				},
			},
		},
		expected: ks(func(ks *v1alpha1.KnativeServing) {
			common.Configure(&ks.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "prometheus")
		}),
		shouldDisableMonitoring: false,
	}, {
		name: "disable monitoring by default",
		in:   &v1alpha1.KnativeServing{},
		expected: ks(func(ks *v1alpha1.KnativeServing) {
			common.Configure(&ks.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
		}),
		shouldDisableMonitoring: true,
	}}
	ctx, _ := injection.EnableInjectionOrDie(context.Background(), nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			objs := c.objs
			if objs == nil {
				objs = []runtime.Object{defaultIngress}
			}
			ks := c.in.DeepCopy()
			ks.Namespace = "knative-serving"
			c.expected.Namespace = ks.Namespace
			ctx, _ = ocpfake.With(ctx, objs...)
			ext := NewExtension(ctx).(*extension)
			ext.kubeclient = fakekubeclientset.NewSimpleClientset([]runtime.Object{&servingNamespace}...)
			if c.shouldDisableMonitoring {
				os.Setenv(monitoring.DisableMonitoringEnvVar, "true")
			}
			ext.Reconcile(context.Background(), ks)
			// Ignore time differences.
			opt := cmp.Comparer(func(apis.VolatileTime, apis.VolatileTime) bool {
				return true
			})
			if !cmp.Equal(ks, c.expected, opt) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", ks, c.expected, cmp.Diff(ks, c.expected, opt))
			}
			if !c.shouldDisableMonitoring {
				ns, err := ext.kubeclient.CoreV1().Namespaces().Get(context.Background(), ks.Namespace, metav1.GetOptions{})
				if err != nil {
					t.Errorf("Namespace %s not found %w", ns, err)
				}
				if len(ns.Labels) != 1 && ns.Labels[monitoring.EnableMonitoringLabel] != "true" {
					t.Errorf("Label is missing for namespace %s ", ks.Namespace)
				}
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
