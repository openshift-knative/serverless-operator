package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func VerifyCRCannotBeInstalledInRandomNamespace(t *testing.T, caCtx *test.Context, namespace string, resource schema.GroupVersionResource, kind string, name string, expectedNamespace string) {
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

	expectedError := fmt.Sprintf("%s may only be created in %s namespace", kind, expectedNamespace)

	if err != nil {
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("expecting the error to contain %q, actual error: %v", expectedError, err)
		}
	} else {
		t.Errorf("it should not be possible to install %s to a test namespace", kind)
	}
}
