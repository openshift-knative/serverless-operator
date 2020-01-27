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

func TestMutate(t *testing.T) {
	const (
		networks = "foo,bar,baz"
		domain   = "fubar"
		image    = "docker.io/queue:tag"
	)
	client := fake.NewFakeClient(mockNetworkConfig(strings.Split(networks, ",")), mockIngressConfig(domain))
	ks := &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving",
			Namespace: "default",
		},
	}
	// Setup image override
	os.Setenv("IMAGE_queue-proxy", image)
	// Mutate for OpenShift
	if err := common.Mutate(ks, client); err != nil {
		t.Error(err)
	}
	verifyEgress(t, ks, networks)
	verifyIngress(t, ks, domain)
	verifyImageOverride(t, ks, image)
	verifyCerts(t, ks)
	verifyTimestamp(t, ks)
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

func verifyEgress(t *testing.T, ks *servingv1alpha1.KnativeServing, expected string) {
	actual := ks.Spec.Config["network"]["istio.sidecar.includeOutboundIPRanges"]
	if actual != expected {
		t.Errorf("Expected '%v', got '%v'", expected, actual)
	}
}

func verifyIngress(t *testing.T, ks *servingv1alpha1.KnativeServing, expected string) {
	domain := ks.Spec.Config["domain"]
	if actual, ok := domain[expected]; !ok || actual != "" {
		t.Errorf("Missing %v, domain=%v", expected, domain)
	}
}

func verifyImageOverride(t *testing.T, ks *servingv1alpha1.KnativeServing, expected string) {
	// Because we overrode the queue image...
	if ks.Spec.Config["deployment"]["queueSidecarImage"] != expected {
		t.Errorf("Missing queue image, config=%v", ks.Spec.Config["deployment"])
	}
	if ks.Spec.Registry.Override["queue-proxy"] != expected {
		t.Errorf("Missing queue image, override=%v", ks.Spec.Registry.Override)
	}
}

func verifyCerts(t *testing.T, ks *servingv1alpha1.KnativeServing) {
	if ks.Spec.ControllerCustomCerts == (servingv1alpha1.CustomCerts{}) {
		t.Error("Missing custom certs config")
	}
}

func verifyTimestamp(t *testing.T, ks *servingv1alpha1.KnativeServing) {
	if _, ok := ks.GetAnnotations()[common.MutationTimestampKey]; !ok {
		t.Error("Missing mutation timestamp annotation")
	}
}
