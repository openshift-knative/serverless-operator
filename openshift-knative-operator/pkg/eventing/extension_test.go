package eventing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/pkg/apis"
	kubefake "knative.dev/pkg/client/injection/kube/client/fake"
	dynamicfake "knative.dev/pkg/injection/clients/dynamicclient/fake"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	ocpfake "github.com/openshift-knative/serverless-operator/pkg/client/injection/client/fake"
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
		in       *operatorv1beta1.KnativeEventing
		expected *operatorv1beta1.KnativeEventing
	}{{
		name:     "all nil",
		in:       &operatorv1beta1.KnativeEventing{},
		expected: ke(istioDisabled),
	}, {
		name: "different HA settings",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					HighAvailability: &base.HighAvailability{
						Replicas: ptr.To(int32(3)),
					},
				},
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			ke.Spec.HighAvailability.Replicas = ptr.To(int32(3))
		}, istioDisabled),
	}, {
		name: "With inclusion sinkbinding setting",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				SinkBindingSelectionMode: "inclusion",
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			ke.Spec.SinkBindingSelectionMode = "inclusion"
		}, istioDisabled),
	}, {
		name: "With exclusion sinkbinding setting",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				SinkBindingSelectionMode: "exclusion",
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			ke.Spec.SinkBindingSelectionMode = "exclusion"
		}, istioDisabled),
	}, {
		name: "With empty sinkbinding setting",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				SinkBindingSelectionMode: "",
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			ke.Spec.SinkBindingSelectionMode = "inclusion"
		}, istioDisabled),
	}, {
		name: "Wrong namespace",
		in: ke(func(ke *operatorv1beta1.KnativeEventing) {
			ke.Namespace = "foo"
		}, istioDisabled),
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			ke.Namespace = "foo"
			ke.Status.MarkInstallFailed(`Knative Eventing must be installed into the namespace "knative-eventing"`)
		}, istioDisabled),
	}, {
		name: "Istio enabled",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Config: base.ConfigMapData{
						"features": map[string]string{
							"istio": "Enabled",
						},
					},
				},
			},
		},
		expected: ke(
			func(eventing *operatorv1beta1.KnativeEventing) {
				if eventing.Spec.Config == nil {
					eventing.Spec.Config = map[string]map[string]string{}
				}
				eventing.Spec.CommonSpec.Config["features"] = map[string]string{
					"istio": "Enabled",
				}
			},
			istioEnabled,
		),
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Default the namespace to the correct one if not set for brevity.
			if c.in.Namespace == "" {
				c.in.Namespace = requiredNs
			}

			ke := c.in.DeepCopy()
			ctx, _ := kubefake.With(context.Background(), &eventingNamespace)
			ctx, _ = dynamicfake.With(ctx, scheme.Scheme)
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

func TestManifests(t *testing.T) {
	cases := []struct {
		name   string
		in     *operatorv1beta1.KnativeEventing
		verify func(manifest []mf.Manifest) error
	}{
		{
			name: "istio enabled",
			in: &operatorv1beta1.KnativeEventing{
				Spec: operatorv1beta1.KnativeEventingSpec{
					CommonSpec: base.CommonSpec{
						Config: base.ConfigMapData{
							"features": map[string]string{
								"istio": "Enabled",
							},
						},
					},
				},
			},
			verify: func(manifest []mf.Manifest) error {
				for _, m := range manifest {
					for _, r := range m.Resources() {
						if r.GroupVersionKind().Kind == "NetworkPolicy" {
							return nil
						}
					}
				}
				return fmt.Errorf("failed to find NetworkPolicy in resources\n%+v", manifest)
			},
		},
		{
			name: "istio enabled, net policies enabled",
			in: &operatorv1beta1.KnativeEventing{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						disableGeneratingIstioNetPoliciesAnnotation: "false", // equivalent to unspecified
					},
				},
				Spec: operatorv1beta1.KnativeEventingSpec{
					CommonSpec: base.CommonSpec{
						Config: base.ConfigMapData{
							"features": map[string]string{
								"istio": "Enabled",
							},
						},
					},
				},
			},
			verify: func(manifest []mf.Manifest) error {
				for _, m := range manifest {
					for _, r := range m.Resources() {
						if r.GroupVersionKind().Kind == "NetworkPolicy" {
							return nil
						}
					}
				}
				return fmt.Errorf("failed to find NetworkPolicy in resources\n%+v", manifest)
			},
		},
		{
			name: "istio enabled, net policies disabled",
			in: &operatorv1beta1.KnativeEventing{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						disableGeneratingIstioNetPoliciesAnnotation: "true", // equivalent to unspecified
					},
				},
				Spec: operatorv1beta1.KnativeEventingSpec{
					CommonSpec: base.CommonSpec{
						Config: base.ConfigMapData{
							"features": map[string]string{
								"istio": "Enabled",
							},
						},
					},
				},
			},
			verify: func(manifest []mf.Manifest) error {
				for _, m := range manifest {
					for _, r := range m.Resources() {
						if r.GroupVersionKind().Kind == "NetworkPolicy" {
							return fmt.Errorf("unexpected network policy %s/%s found", r.GetNamespace(), r.GetName())
						}
					}
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, filename, _, _ := goruntime.Caller(0)
			filename = filepath.Join(filepath.Dir(filename), "..", "monitoring", "testdata", "rbac-proxy.yaml")
			t.Setenv("SERVICE_MONITOR_RBAC_MANIFEST_PATH", filename)
			if tc.in.Namespace == "" {
				tc.in.Namespace = requiredNs
			}
			ke := tc.in
			ctx, _ := kubefake.With(context.Background(), &eventingNamespace)
			ctx, _ = dynamicfake.With(ctx, scheme.Scheme)
			ext := NewExtension(ctx, nil)
			m, err := ext.Manifests(ke)
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.verify(m); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestMonitoring(t *testing.T) {
	cases := []struct {
		name     string
		in       *operatorv1beta1.KnativeEventing
		expected *operatorv1beta1.KnativeEventing
		// Returns the expected status for monitoring
		setupMonitoringToggle func() (bool, error)
	}{{
		name:                  "enable monitoring when monitoring toggle is not defined, backend is not defined",
		in:                    &operatorv1beta1.KnativeEventing{},
		expected:              ke(istioDisabled),
		setupMonitoringToggle: func() (bool, error) { return true, nil },
	}, {
		name: "enable monitoring when monitoring toggle = not defined, backend = defined and not `none`",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "prometheus"}},
				},
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "prometheus")
		}, istioDisabled),
		setupMonitoringToggle: func() (bool, error) { return true, nil },
	}, {
		name: "disable monitoring when monitoring toggle is not defined, backend is `none`",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "none"}},
				},
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
		}, istioDisabled),
		setupMonitoringToggle: func() (bool, error) { return false, nil },
	}, {
		name:                  "enable monitoring when monitoring toggle is on, backend is not defined",
		in:                    &operatorv1beta1.KnativeEventing{},
		expected:              ke(istioDisabled),
		setupMonitoringToggle: func() (bool, error) { return true, os.Setenv(monitoring.EnableMonitoringEnvVar, "true") },
	}, {
		name: "enable monitoring when monitoring toggle is on, backend is defined and not `none`",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "prometheus"}},
				},
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "prometheus")
		}, istioDisabled),
		setupMonitoringToggle: func() (bool, error) {
			return true, os.Setenv(monitoring.EnableMonitoringEnvVar, "true")
		},
	}, {
		name: "disable monitoring when monitoring toggle is on, backend is `none`",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "none"}},
				},
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
		}, istioDisabled),
		setupMonitoringToggle: func() (bool, error) {
			return false, os.Setenv(monitoring.EnableMonitoringEnvVar, "true")
		},
	}, {
		name: "disable monitoring when monitoring toggle is off, backend is not defined",
		in:   &operatorv1beta1.KnativeEventing{},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
		}, istioDisabled),
		setupMonitoringToggle: func() (bool, error) { return false, os.Setenv(monitoring.EnableMonitoringEnvVar, "false") },
	}, {
		name: "enable monitoring when monitoring toggle = off, backend = defined and not `none`",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "prometheus"}},
				},
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "prometheus")
		}, istioDisabled),
		setupMonitoringToggle: func() (bool, error) { return true, os.Setenv(monitoring.EnableMonitoringEnvVar, "false") },
	}, {
		name: "disable monitoring when monitoring toggle is off, backend is `none`",
		in: &operatorv1beta1.KnativeEventing{
			Spec: operatorv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{monitoring.ObservabilityCMName: {monitoring.ObservabilityBackendKey: "none"}},
				},
			},
		},
		expected: ke(func(ke *operatorv1beta1.KnativeEventing) {
			common.Configure(&ke.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
		}, istioDisabled),
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
			ctx, _ = dynamicfake.With(ctx, scheme.Scheme)
			ext := NewExtension(ctx, nil)
			shouldEnableMonitoring, err := c.setupMonitoringToggle()

			if err != nil {
				t.Errorf("Failed to setup the monitoring toggle %v", err)
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

func ke(mods ...func(*operatorv1beta1.KnativeEventing)) *operatorv1beta1.KnativeEventing {
	base := &operatorv1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: requiredNs,
		},
		Spec: operatorv1beta1.KnativeEventingSpec{
			SinkBindingSelectionMode: "inclusion",
			CommonSpec: base.CommonSpec{
				HighAvailability: &base.HighAvailability{
					Replicas: ptr.To(int32(2)),
				},
				Registry: base.Registry{
					Default: "bar2",
					Override: map[string]string{
						"default": "bar2",
						"foo":     "bar",
					},
				},
				DeprecatedResources: []base.ResourceRequirementsOverride{{
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

func istioDisabled(ke *operatorv1beta1.KnativeEventing) {
	ke.Spec.Workloads = append(ke.Spec.Workloads, base.WorkloadOverride{
		Name:     "eventing-istio-controller",
		Replicas: ptr.To(int32(0)),
	})
}

func istioEnabled(ke *operatorv1beta1.KnativeEventing) {
	ke.Spec.Workloads = append(ke.Spec.Workloads, base.WorkloadOverride{
		Name:     "eventing-istio-controller",
		Replicas: ptr.To(int32(1)),
	})
}
