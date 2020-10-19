package common_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
}

func newKs() *servingv1alpha1.KnativeServing {
	return &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving",
			Namespace: namespace,
		},
	}
}

func TestMutate(t *testing.T) {
	const (
		networks = "foo,bar,baz"
		domain   = "fubar"
		image    = "docker.io/queue:tag"
	)
	os.Setenv("IMAGE_SERVING_queue-proxy", image)
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

		client := fake.NewFakeClient(mockNetworkConfig(strings.Split(networks, ",")), mockIngressConfig(domain))
		// Setup image override
		// Mutate for OpenShift
		err := common.Mutate(ks, client)
		if err != nil {
			t.Error(err)
		}

		verifyNetworkConfig(t, ks)
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
		verifyNetworkConfig(t, ks)
		verifyIngress(t, ks, domain)
		verifyImageOverride(t, &ks.Spec.Registry, "queue-proxy", image)
		verifyQueueProxySidecarImageOverride(t, ks, image)
		verifyCerts(t, ks)
		verifyWebookMemoryLimit(t, ks)
		tc.ha(t, ks)

		// Force a change and rerun
		ks.Spec.Config["network"]["ingress.class"] = "foo"
		ks.Spec.Config["network"]["domainTemplate"] = "{{.Name}}.{{.Namespace}}.{{Domain}}"
		err = common.Mutate(ks, client)
		if err != nil {
			t.Error(err)
		}
		verifyNetworkConfig(t, ks)
		verifyIngress(t, ks, domain)
		verifyImageOverride(t, &ks.Spec.Registry, "queue-proxy", image)
		verifyQueueProxySidecarImageOverride(t, ks, image)
		verifyCerts(t, ks)
		verifyWebookMemoryLimit(t, ks)
		tc.ha(t, ks)
	}
}

func TestWebhookMemoryLimit(t *testing.T) {
	var testdata = []byte(`
- input:
    apiVersion: operator.knative.dev/v1alpha1
    kind: KnativeServing
    metadata:
      name: no-overrides
  expected:
  - container: webhook
    limits:
      memory: 1024Mi
- input:
    apiVersion: operator.knative.dev/v1alpha1
    kind: KnativeServing
    metadata:
      name: add-webhook-to-existing-override
    spec:
      resources:
      - container: activator
        limits:
          cpu: 9999m
          memory: 999Mi
  expected:
  - container: activator
    limits:
      cpu: 9999m
      memory: 999Mi
  - container: webhook
    limits:
      memory: 1024Mi
- input:
    apiVersion: operator.knative.dev/v1alpha1
    kind: KnativeServing
    metadata:
      name: preserve-webhook-values
    spec:
      resources:
      - container: webhook
        requests:
          cpu: 22m
          memory: 22Mi
        limits:
          cpu: 220m
          memory: 220Mi
  expected:
  - container: webhook
    requests:
      cpu: 22m
      memory: 22Mi
    limits:
      cpu: 220m
      memory: 220Mi
`)
	tests := []struct {
		Input    servingv1alpha1.KnativeServing
		Expected []servingv1alpha1.ResourceRequirementsOverride
	}{}
	if err := yaml.Unmarshal(testdata, &tests); err != nil {
		t.Fatalf("Failed to unmarshal tests: %v", err)
	}
	client := fake.NewFakeClient(mockIngressConfig("whatever"))
	for _, test := range tests {
		t.Run(test.Input.Name, func(t *testing.T) {
			err := common.Mutate(&test.Input, client)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(test.Input.Spec.Resources, test.Expected) {
				t.Errorf("\n    Name: %s\n  Expect: %v\n  Actual: %v", test.Input.Name, test.Expected, test.Input.Spec.Resources)
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

func verifyNetworkConfig(t *testing.T, ks *servingv1alpha1.KnativeServing) {
	network := ks.Spec.Config["network"]

	if actual := network["domainTemplate"]; actual != common.DefaultDomainTemplate {
		t.Errorf("got %q, want %q", actual, common.DefaultDomainTemplate)
	}

	if actual := network["ingress.class"]; actual != common.DefaultIngressClass {
		t.Errorf("got %q, want %q", actual, common.DefaultIngressClass)
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
