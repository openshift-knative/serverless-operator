package common_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestMutateEventing(t *testing.T) {
	const (
		image1 = "quay.io/foo:tag"
		image2 = "quay.io/baz:tag"
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
	tests := []struct {
		name string
		in   []operatorv1alpha1.ResourceRequirementsOverride
		want []operatorv1alpha1.ResourceRequirementsOverride
	}{{
		name: "no overrides",
		in:   nil,
		want: []operatorv1alpha1.ResourceRequirementsOverride{{
			Container: "eventing-webhook",
			ResourceRequirements: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("1024Mi"),
				},
			},
		}},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			obj := &operatorv1alpha1.KnativeEventing{
				Spec: operatorv1alpha1.KnativeEventingSpec{
					CommonSpec: operatorv1alpha1.CommonSpec{
						Resources: test.in,
					},
				},
			}

			common.MutateEventing(obj)
			if !cmp.Equal(obj.Spec.Resources, test.want, cmpopts.IgnoreUnexported(resource.Quantity{})) {
				t.Errorf("Resources not as expected, diff: %s", cmp.Diff(test.want, obj.Spec.Resources))
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
