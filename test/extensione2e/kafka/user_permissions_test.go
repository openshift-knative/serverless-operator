package knativekafkae2e

import (
	"testing"

	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/tracker"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	kafkabindingsv1beta1 "knative.dev/eventing-kafka/pkg/apis/bindings/v1beta1"
)

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

	tests := []test.UserPermissionTest{{
		Name:        "project admin user",
		UserContext: paCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			kafkaBindingsGVR: test.AllowAll,
		},
	}, {
		Name:        "edit user",
		UserContext: editCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			kafkaBindingsGVR: test.AllowAll,
		},
	}, {
		Name:        "view user",
		UserContext: viewCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			kafkaBindingsGVR: test.AllowViewOnly,
		},
	}}

	test.RunUserPermissionTests(t, objects, tests...)
}
