package knativekafkae2e

import (
	"context"
	"testing"

	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/tracker"

	"github.com/openshift-knative/serverless-operator/test"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	kafkabindingsv1beta1 "knative.dev/eventing-kafka/pkg/apis/bindings/v1beta1"
)

type allowedOperations struct {
	get    bool
	list   bool
	create bool
	delete bool
}

func init() {
	kafkabindingsv1beta1.AddToScheme(scheme.Scheme)
}

func TestKafkaUserPermissions(t *testing.T) {
	paCtx := test.SetupProjectAdmin(t)
	editCtx := test.SetupEdit(t)
	viewCtx := test.SetupView(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, paCtx, editCtx, viewCtx) })
	defer test.CleanupAll(t, paCtx, editCtx, viewCtx)

	kafkaBindingsGVR := kafkabindingsv1beta1.SchemeGroupVersion.WithResource("kafkabindings")

	kafkaBinding := &kafkabindingsv1beta1.KafkaBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-kafka-binding",
		},
		Spec: kafkabindingsv1beta1.KafkaBindingSpec{
			KafkaAuthSpec: kafkabindingsv1beta1.KafkaAuthSpec{
				BootstrapServers: []string{"myserver:9092"},
			},
			BindingSpec: duckv1alpha1.BindingSpec{
				Subject: tracker.Reference{
					APIVersion: "batch/v1",
					Kind:       "Job",
					Name:       "my-job",
				},
			},
		},
	}

	objects := map[schema.GroupVersionResource]*unstructured.Unstructured{
		kafkaBindingsGVR: {},
	}

	if err := scheme.Scheme.Convert(kafkaBinding, objects[kafkaBindingsGVR], nil); err != nil {
		t.Fatalf("Failed to convert KafkaBinding: %v", err)
	}

	allowAll := allowedOperations{
		get:    true,
		list:   true,
		create: true,
		delete: true,
	}
	allowViewOnly := allowedOperations{
		get:  true,
		list: true,
	}

	tests := []struct {
		name        string
		userContext *test.Context
		allowed     map[schema.GroupVersionResource]allowedOperations
	}{{
		name:        "project admin user",
		userContext: paCtx,
		allowed: map[schema.GroupVersionResource]allowedOperations{
			kafkaBindingsGVR: allowAll,
		},
	}, {
		name:        "edit user",
		userContext: editCtx,
		allowed: map[schema.GroupVersionResource]allowedOperations{
			kafkaBindingsGVR: allowAll,
		},
	}, {
		name:        "view user",
		userContext: viewCtx,
		allowed: map[schema.GroupVersionResource]allowedOperations{
			kafkaBindingsGVR: allowViewOnly,
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for gvr, allowed := range test.allowed {
				client := test.userContext.Clients.Dynamic.Resource(gvr).Namespace(testNamespace)

				obj := objects[gvr].DeepCopy()
				obj.SetName("test-" + gvr.Resource)

				_, err := client.Create(context.Background(), obj, metav1.CreateOptions{})
				if (allowed.create && err != nil) || (!allowed.create && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error creating %s, allowed = %v, err = %v", gvr.String(), allowed.create, err)
				}

				err = client.Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
				if (allowed.delete && err != nil) || (!allowed.delete && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error deleting %s, allowed = %v, err = %v", gvr.String(), allowed.delete, err)
				}

				_, err = client.Get(context.Background(), obj.GetName(), metav1.GetOptions{})
				// Ignore IsNotFound errors as "Forbidden" would overrule it anyway.
				if (allowed.get && err != nil && !apierrs.IsNotFound(err)) || (!allowed.get && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error getting %s, allowed = %v, err = %v", gvr.String(), allowed.get, err)
				}

				_, err = client.List(context.Background(), metav1.ListOptions{})
				if (allowed.list && err != nil) || (!allowed.list && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error listing %s, allowed = %v, err = %v", gvr.String(), allowed.list, err)
				}
			}
		})
	}
}
