package common_test

import (
	"os"
	"reflect"
	"sigs.k8s.io/yaml"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
}

func TestMutateEventing(t *testing.T) {
	const (
		image1 = "docker.io/foo:tag"
		image2 = "docker.io/baz:tag"
	)
	client := fake.NewFakeClient()
	ke := &operatorv1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-eventing",
			Namespace: "default",
		},
	}
	// Setup image override
	os.Setenv("IMAGE_foo", image1)
	// Setup image override with deployment name
	os.Setenv("IMAGE_bar__baz", image2)

	// Mutate for OpenShift
	if err := common.MutateEventing(ke, client); err != nil {
		t.Error(err)
	}
	verifyImageOverride(t, &ke.Spec.Registry, "foo", image1)
	verifyImageOverride(t, &ke.Spec.Registry, "bar/baz", image2)
}

func TestEventingWebhookMemoryLimit(t *testing.T) {
	var testdata = []byte(`
- input:
    apiVersion: operator.knative.dev/v1alpha1
    kind: KnativeEventing
    metadata:
      name: no-overrides
  expected:
  - container: eventing-webhook
    limits:
      memory: 1024Mi
`)
	tests := []struct {
		Input    operatorv1alpha1.KnativeEventing
		Expected []operatorv1alpha1.ResourceRequirementsOverride
	}{}
	if err := yaml.Unmarshal(testdata, &tests); err != nil {
		t.Fatalf("Failed to unmarshal tests: %v", err)
	}
	client := fake.NewFakeClient(mockIngressConfig("whatever"))
	for _, test := range tests {
		t.Run(test.Input.Name, func(t *testing.T) {
			err := common.MutateEventing(&test.Input, client)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(test.Input.Spec.Resources, test.Expected) {
				t.Errorf("\n    Name: %s\n  Expect: %v\n  Actual: %v", test.Input.Name, test.Expected, test.Input.Spec.Resources)
			}
		})
	}
}
