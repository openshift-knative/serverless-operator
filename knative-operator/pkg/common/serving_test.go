package common_test

import (
	"os"
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
}

func newKs() *servingv1alpha1.KnativeServing {
	return &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      servingName,
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

		client := fake.NewFakeClient(mockNetworkConfig(strings.Split(networks, ",")), mockIngressConfig(domain))
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
		tc.ha(t, ks)

		// Force a change and rerun
		ks.Spec.Config["network"]["ingress.class"] = "foo"
		err = common.Mutate(ks, client)
		if err != nil {
			t.Error(err)
		}
		verifyIngress(t, ks, domain)
		verifyImageOverride(t, &ks.Spec.Registry, "queue-proxy", image)
		verifyQueueProxySidecarImageOverride(t, ks, image)
		verifyCerts(t, ks)
		tc.ha(t, ks)
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
