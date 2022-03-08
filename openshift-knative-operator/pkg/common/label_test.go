package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestInjectCommonLabelIntoNamespace(t *testing.T) {
	tests := []struct {
		name string
		in   *unstructured.Unstructured
		want string
	}{{
		name: "inject common label into namespace",
		in: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Namespace",
				"metadata": map[string]interface{}{
					"name": "test",
				},
			},
		},
		want: "openshift-serverless",
	}, {
		name: "do not inject common label into deployment",
		in: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name": "test",
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u := test.in
			if err := InjectCommonLabelIntoNamespace()(u); err != nil {
				t.Fatal("Unexpected error from transformer", err)
			}

			if !cmp.Equal(u.GetLabels()["knative.openshift.io/part-of"], test.want) {
				t.Errorf("Unexpected label: Got = %q, want = %q", u.GetLabels()["knative.openshift.io/part-of"], test.want)
			}
		})
	}
}
