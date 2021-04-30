package common_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/yaml"
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
	os.Setenv("IMAGE_foo", image1)
	// Setup image override with deployment name
	os.Setenv("IMAGE_bar__baz", image2)

	// Mutate for OpenShift
	common.MutateEventing(ke)
	verifyEventingHA(t, ke, 2)
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
			if !cmp.Equal(test.Input.Spec.Resources, test.Expected, cmpopts.IgnoreUnexported(resource.Quantity{})) {
				t.Errorf("Resources not as expected, diff: %s", cmp.Diff(test.Expected, test.Input.Spec.Resources))
			}
		})
	}
}

func TestEventingWebhookInclusionMode(t *testing.T) {

	tests := []struct {
		name   string
		ke     *operatorv1alpha1.KnativeEventing
		wanted string
	}{
		{
			name: "No mode specified",
			ke: &operatorv1alpha1.KnativeEventing{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-mode-specified",
				},
			},
			wanted: "inclusion",
		},
		{
			name: "Inclusion Mode Specified",
			ke: &operatorv1alpha1.KnativeEventing{
				ObjectMeta: metav1.ObjectMeta{
					Name: "inclusion-specified",
				},
				Spec: operatorv1alpha1.KnativeEventingSpec{
					SinkBindingSelectionMode: "inclusion",
				},
			},
			wanted: "inclusion",
		},
		{
			name: "Exclusion Mode Specified",
			ke: &operatorv1alpha1.KnativeEventing{
				ObjectMeta: metav1.ObjectMeta{
					Name: "exclusion-specified",
				},
				Spec: operatorv1alpha1.KnativeEventingSpec{
					SinkBindingSelectionMode: "exclusion",
				},
			},
			wanted: "exclusion",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			common.MutateEventing(tc.ke)
			if tc.ke.Spec.SinkBindingSelectionMode != tc.wanted {
				t.Errorf(`Name: %s\n Expected "%s", Got: "%s"`, tc.name, tc.wanted, tc.ke.Spec.SinkBindingSelectionMode)
			}
		})
	}
}

func verifyEventingHA(t *testing.T, ke *operatorv1alpha1.KnativeEventing, replicas int32) {
	if ke.Spec.HighAvailability == nil {
		t.Error("Missing HA")
		return
	}

	if ke.Spec.HighAvailability.Replicas != replicas {
		t.Errorf("Wrong ha replica size: expected%v, got %v", replicas, ke.Spec.HighAvailability.Replicas)
	}
}
