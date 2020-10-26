package e2e

import (
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const gcr = "gcr.io"

func VerifyNoDisallowedImageReference(t *testing.T, caCtx *test.Context, namespace string) {
	podSpecableTypes := []schema.GroupVersionResource{{
		Group:    "apps",
		Version:  "v1",
		Resource: "daemonsets",
	}, {
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	}, {
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}, {
		Group:    "apps",
		Version:  "v1",
		Resource: "replicasets",
	}}

	for _, podSpecableType := range podSpecableTypes {

		result, err := caCtx.Clients.Dynamic.Resource(podSpecableType).Namespace(namespace).List(metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Error listing %v: %v", podSpecableType, err)
		}

		for _, u := range result.Items {
			ps := &duckv1.WithPod{}

			err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), ps)
			if err != nil {
				t.Fatalf("Error converting unstructured to podSpecable %v: %v", u, err)
			}

			for _, container := range ps.Spec.Template.Spec.Containers {
				if strings.Contains(container.Image, gcr) {
					t.Fatalf("Container %q in resource %q of type %v contains disallowed image registry ref %q in image %q", container.Name, ps.Name, podSpecableType, gcr, container.Image)
				}

				for _, env := range container.Env {
					if strings.Contains(env.Value, gcr) {
						t.Fatalf("Container %q in resource %q of type %v contains disallowed image registry ref %q in env var %q", container.Name, ps.Name, podSpecableType, gcr, env.Name)
					}
				}
			}
		}

	}

}
