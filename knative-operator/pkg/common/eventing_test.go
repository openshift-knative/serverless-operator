package common_test

import (
	"os"
	"reflect"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestMutateEventing(t *testing.T) {
	const (
		image1 = "docker.io/foo:tag"
		image2 = "docker.io/baz:tag"
	)
	ke := &operatorv1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-eventing",
			Namespace: "default",
		},
	}
	// Setup image override
	os.Setenv("IMAGE_EVENTING_foo", image1)
	// Setup image override with deployment name
	os.Setenv("IMAGE_EVENTING_bar__baz", image2)

	// Mutate for OpenShift
	common.MutateEventing(ke)
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
	for _, test := range tests {
		t.Run(test.Input.Name, func(t *testing.T) {
			common.MutateEventing(&test.Input)
			if !reflect.DeepEqual(test.Input.Spec.Resources, test.Expected) {
				t.Errorf("\n    Name: %s\n  Expect: %v\n  Actual: %v", test.Input.Name, test.Expected, test.Input.Spec.Resources)
			}
		})
	}
}
