package common_test

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	apis.AddToScheme(scheme.Scheme)
}

func newKs() *servingv1alpha1.KnativeServing {
	return &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name: "knative-serving",
		},
	}
}

func TestMutate(t *testing.T) {
	const (
		networks = "foo,bar,baz"
		domain   = "fubar"
		image    = "quay.io/queue:tag"
	)
	os.Setenv("IMAGE_queue-proxy", image)
	type check func(*testing.T, *servingv1alpha1.KnativeServing)

	cases := []struct {
		name string
		ks   *servingv1alpha1.KnativeServing
		ha   check
	}{
		{
			name: "HA defaulted",
			ks:   newKs(),
			ha:   verifyDefaultHA,
		},
		{
			name: "HA not defaulted",
			ks: func() *servingv1alpha1.KnativeServing {
				s := newKs()
				s.Spec.HighAvailability = &servingv1alpha1.HighAvailability{
					Replicas: 1,
				}

				return s
			}(),
			ha: verifyOverriddenHA,
		},
	}

	for i := range cases {
		tc := cases[i]
		ks := tc.ks

		client := fake.NewClientBuilder().
			WithObjects(mockNetworkConfig(strings.Split(networks, ",")), mockIngressConfig(domain)).
			Build()
		// Setup image override
		// Mutate for OpenShift
		err := common.Mutate(ks, client)
		if err != nil {
			t.Error(err)
		}

		verifyIngress(t, ks, domain)
		verifyImageOverride(t, &ks.Spec.Registry, "queue-proxy", image)
		verifyQueueProxySidecarImageOverride(t, ks, image)
		verifyCerts(t, ks)
		verifyWebookMemoryLimit(t, ks)
		tc.ha(t, ks)

		// Rerun, should be a noop
		err = common.Mutate(ks, client)
		if err != nil {
			t.Error(err)
		}
		verifyIngress(t, ks, domain)
		verifyImageOverride(t, &ks.Spec.Registry, "queue-proxy", image)
		verifyQueueProxySidecarImageOverride(t, ks, image)
		verifyCerts(t, ks)
		verifyWebookMemoryLimit(t, ks)
		tc.ha(t, ks)

		// Force a change and rerun
		ks.Spec.Config["network"] = map[string]string{
			"ingress.class":  "foo",
			"domainTemplate": "{{.Name}}.{{.Namespace}}.{{Domain}}",
		}
		err = common.Mutate(ks, client)
		if err != nil {
			t.Error(err)
		}
		verifyIngress(t, ks, domain)
		verifyImageOverride(t, &ks.Spec.Registry, "queue-proxy", image)
		verifyQueueProxySidecarImageOverride(t, ks, image)
		verifyCerts(t, ks)
		verifyWebookMemoryLimit(t, ks)
		tc.ha(t, ks)
	}
}

func TestWebhookMemoryLimit(t *testing.T) {
	tests := []struct {
		name string
		in   []servingv1alpha1.ResourceRequirementsOverride
		want []servingv1alpha1.ResourceRequirementsOverride
	}{{
		name: "no overrides",
		in:   nil,
		want: []servingv1alpha1.ResourceRequirementsOverride{{
			Container: "webhook",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("1024Mi"),
				},
			},
		}},
	}, {
		name: "add webhook to existing override",
		in: []servingv1alpha1.ResourceRequirementsOverride{{
			Container: "activator",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("99Mi"),
				},
			},
		}},
		want: []servingv1alpha1.ResourceRequirementsOverride{{
			Container: "activator",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("99Mi"),
				},
			},
		}, {
			Container: "webhook",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("1024Mi"),
				},
			},
		}},
	}, {
		name: "preserve webhook values",
		in: []servingv1alpha1.ResourceRequirementsOverride{{
			Container: "webhook",
			ResourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("22m"),
					corev1.ResourceMemory: resource.MustParse("22Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("220m"),
					corev1.ResourceMemory: resource.MustParse("220Mi"),
				},
			},
		}},
		want: []servingv1alpha1.ResourceRequirementsOverride{{
			Container: "webhook",
			ResourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("22m"),
					corev1.ResourceMemory: resource.MustParse("22Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("220m"),
					corev1.ResourceMemory: resource.MustParse("220Mi"),
				},
			},
		}},
	}}
	client := fake.NewClientBuilder().WithObjects(mockIngressConfig("whatever")).Build()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			obj := &servingv1alpha1.KnativeServing{
				Spec: servingv1alpha1.KnativeServingSpec{
					CommonSpec: servingv1alpha1.CommonSpec{
						Resources: test.in,
					},
				},
			}
			err := common.Mutate(obj, client)
			if err != nil {
				t.Error(err)
			}
			if !cmp.Equal(obj.Spec.Resources, test.want, cmpopts.IgnoreUnexported(resource.Quantity{})) {
				t.Errorf("Resources not as expected, diff: %s", cmp.Diff(test.want, obj.Spec.Resources))
			}
		})
	}
}

func mockNetworkConfig(networks []string) *configv1.Network {
	return &configv1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.NetworkSpec{
			ServiceNetwork: networks,
		},
	}
}

func mockIngressConfig(domain string) *configv1.Ingress {
	return &configv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.IngressSpec{
			Domain: domain,
		},
	}
}

func verifyIngress(t *testing.T, ks *servingv1alpha1.KnativeServing, expected string) {
	domain := ks.Spec.Config["domain"]
	if actual, ok := domain[expected]; !ok || actual != "" {
		t.Errorf("Missing %v, domain=%v", expected, domain)
	}
}

func verifyQueueProxySidecarImageOverride(t *testing.T, ks *servingv1alpha1.KnativeServing, expected string) {
	// Because we overrode the queue image...
	if ks.Spec.Config["deployment"]["queueSidecarImage"] != expected {
		t.Errorf("Missing queue image, config=%v", ks.Spec.Config["deployment"])
	}
}

func verifyCerts(t *testing.T, ks *servingv1alpha1.KnativeServing) {
	if ks.Spec.ControllerCustomCerts == (servingv1alpha1.CustomCerts{}) {
		t.Error("Missing custom certs config")
	}
}

func verifyWebookMemoryLimit(t *testing.T, ks *servingv1alpha1.KnativeServing) {
	for _, v := range ks.Spec.Resources {
		if v.Container == "webhook" {
			if _, ok := v.Limits[corev1.ResourceMemory]; ok {
				return
			}
		}
	}
	t.Error("Missing webhook memory limit")
}

func verifyDefaultHA(t *testing.T, ks *servingv1alpha1.KnativeServing) {
	verifyHA(t, ks, 2)
}

func verifyOverriddenHA(t *testing.T, ks *servingv1alpha1.KnativeServing) {
	verifyHA(t, ks, 1)
}

func verifyHA(t *testing.T, ks *servingv1alpha1.KnativeServing, replicas int32) {
	if ks.Spec.HighAvailability == nil {
		t.Error("Missing HA")
		return
	}

	if ks.Spec.HighAvailability.Replicas != replicas {
		t.Errorf("Wrong ha replica size: expected%v, got %v", replicas, ks.Spec.HighAvailability.Replicas)
	}
}
