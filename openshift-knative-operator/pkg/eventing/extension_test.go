package eventing

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	ocpfake "github.com/openshift-knative/serverless-operator/pkg/client/injection/client/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func init() {
	os.Setenv("IMAGE_foo", "bar")
	os.Setenv("IMAGE_default", "bar2")
	os.Setenv(requiredNsEnvName, requiredNs)
}

func TestReconcile(t *testing.T) {

	cases := []struct {
		name     string
		in       *v1alpha1.KnativeEventing
		expected *v1alpha1.KnativeEventing
	}{{
		name:     "all nil",
		in:       &v1alpha1.KnativeEventing{},
		expected: ke(),
	}, {
		name: "different HA settings",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					HighAvailability: &v1alpha1.HighAvailability{
						Replicas: 3,
					},
				},
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			ke.Spec.HighAvailability.Replicas = 3
		}),
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

			ke := c.in.DeepCopy()
			ctx, _ := kubefake.With(context.Background(), &eventingNamespace)
			ext := NewExtension(ctx, nil)
			ext.Reconcile(context.Background(), ke)

			// Ignore time differences.
			opt := cmp.Comparer(func(apis.VolatileTime, apis.VolatileTime) bool {
				return true
			})

			if !cmp.Equal(ke, c.expected, opt) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", ke, c.expected, cmp.Diff(ke, c.expected, opt))
			}
		})
	}
}

func TestMonitoring(t *testing.T) {
	cases := []struct {
		name     string
		in       *v1alpha1.KnativeEventing
		expected *v1alpha1.KnativeEventing
		// Returns the expected status for monitoring
		setupMonitoringToggle func() (bool, error)
	}{{
		name:                  "enable monitoring when monitoring toggle is not defined, backend is not defined",
		in:                    &v1alpha1.KnativeEventing{},
		expected:              ke(),
		setupMonitoringToggle: func() (bool, error) { return true, nil },
	}, {
		name: "enable monitoring when monitoring toggle = not defined, backend = defined and not `none`",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "prometheus"}},
				},
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "prometheus")
		}),
		setupMonitoringToggle: func() (bool, error) { return true, nil },
	}, {
		name: "disable monitoring when monitoring toggle is not defined, backend is `none`",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "none"}},
				},
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
		}),
		setupMonitoringToggle: func() (bool, error) { return false, nil },
	}, {
		name:                  "enable monitoring when monitoring toggle is on, backend is not defined",
		in:                    &v1alpha1.KnativeEventing{},
		expected:              ke(),
		setupMonitoringToggle: func() (bool, error) { return true, os.Setenv(monitoring.EnableMonitoringEnvVar, "true") },
	}, {
		name: "enable monitoring when monitoring toggle is on, backend is defined and not `none`",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "prometheus"}},
				},
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "prometheus")
		}),
		setupMonitoringToggle: func() (bool, error) {
			return true, os.Setenv(monitoring.EnableMonitoringEnvVar, "true")
		},
	}, {
		name: "disable monitoring when monitoring toggle is on, backend is `none`",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "none"}},
				},
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
		}),
		setupMonitoringToggle: func() (bool, error) {
			return false, os.Setenv(monitoring.EnableMonitoringEnvVar, "true")
		},
	}, {
		name: "disable monitoring when monitoring toggle is off, backend is not defined",
		in:   &v1alpha1.KnativeEventing{},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
		}),
		setupMonitoringToggle: func() (bool, error) { return false, os.Setenv(monitoring.EnableMonitoringEnvVar, "false") },
	}, {
		name: "enable monitoring when monitoring toggle = off, backend = defined and not `none`",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "prometheus"}},
				},
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "prometheus")
		}),
		setupMonitoringToggle: func() (bool, error) { return true, os.Setenv(monitoring.EnableMonitoringEnvVar, "false") },
	}, {
		name: "disable monitoring when monitoring toggle is off, backend is `none`",
		in: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "none"}},
				},
			},
		},
		expected: ke(func(ke *v1alpha1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
		}),
		setupMonitoringToggle: func() (bool, error) { return false, os.Setenv(monitoring.EnableMonitoringEnvVar, "false") },
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			objs := []runtime.Object{&eventingNamespace}
			ke := c.in.DeepCopy()
			if ke.Namespace == "" {
				ke.Namespace = requiredNs
			}
			c.expected.Namespace = ke.Namespace
			ctx, _ := ocpfake.With(context.Background(), objs...)
			ctx, kube := kubefake.With(ctx, &eventingNamespace)
			ext := NewExtension(ctx, nil)
			shouldEnableMonitoring, err := c.setupMonitoringToggle()

			if err != nil {
				t.Errorf("Failed to setup the monitoring toggle %w", err)
			}
			ext.Reconcile(context.Background(), ke)

			// Ignore time differences.
			opt := cmp.Comparer(func(apis.VolatileTime, apis.VolatileTime) bool {
				return true
			})
			if !cmp.Equal(ke, c.expected, opt) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", ke, c.expected, cmp.Diff(ke, c.expected, opt))
			}
			ns, err := kube.CoreV1().Namespaces().Get(context.Background(), ke.Namespace, metav1.GetOptions{})
			if err != nil {
				t.Errorf("Failed to get namespace %s: %v", ns, err)
			}
			if ns.Labels[monitoring.EnableMonitoringLabel] != strconv.FormatBool(shouldEnableMonitoring) {
				t.Errorf("Label is missing for namespace %s ", ke.Namespace)
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
				HighAvailability: &v1alpha1.HighAvailability{
					Replicas: 2,
				},
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
