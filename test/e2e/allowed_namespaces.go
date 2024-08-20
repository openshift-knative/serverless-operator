package e2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func VerifyCRCannotBeInstalledInRandomNamespace(t *testing.T, caCtx *test.Context, namespace string, resource schema.GroupVersionResource, kind string, name string) {
	dynamicClient := caCtx.Clients.Dynamic.Resource(resource).Namespace(namespace)

	// Attempt to install the resource to the test namespace
	_, err := dynamicClient.Create(context.Background(), &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": resource.GroupVersion().String(),
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}, metav1.CreateOptions{})

	if err != nil {
		t.Logf("actual error creating %s: %v", kind, err)
	}

	if err == nil {
		t.Errorf("It should not be possible to install %s to a test namespace", kind)
	}
}
