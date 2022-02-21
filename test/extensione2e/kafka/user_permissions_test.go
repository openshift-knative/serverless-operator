package knativekafkae2e

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	kafkabindingv1beta1 "knative.dev/eventing-kafka/pkg/apis/bindings/v1beta1"
	kafkasourcesv1beta1 "knative.dev/eventing-kafka/pkg/apis/sources/v1beta1"
)

func init() {
	kafkasourcesv1beta1.AddToScheme(scheme.Scheme)
}

func TestKafkaUserPermissions(t *testing.T) {
	paCtx := test.SetupProjectAdmin(t)
	editCtx := test.SetupEdit(t)
	viewCtx := test.SetupView(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, paCtx, editCtx, viewCtx) })
	defer test.CleanupAll(t, paCtx, editCtx, viewCtx)

	kafkaSourcesGVR := kafkasourcesv1beta1.SchemeGroupVersion.WithResource("kafkasources")

	kafkaSource := &kafkasourcesv1beta1.KafkaSource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-kafka-binding",
		},
		Spec: kafkasourcesv1beta1.KafkaSourceSpec{
			KafkaAuthSpec: kafkabindingv1beta1.KafkaAuthSpec{
				BootstrapServers: []string{"myserver:9092"},
			},
			Topics:        []string{"my-topic"},
			ConsumerGroup: "my-cg",
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: ksvcAPIVersion,
						Kind:       ksvcKind,
						Name:       "fakeKSVC",
					},
				},
			},
		},
	}

	objects := map[schema.GroupVersionResource]*unstructured.Unstructured{
		kafkaSourcesGVR: {},
	}

	if err := scheme.Scheme.Convert(kafkaSource, objects[kafkaSourcesGVR], nil); err != nil {
		t.Fatalf("Failed to convert KafkaSource: %v", err)
	}

	tests := []test.UserPermissionTest{{
		Name:        "project admin user",
		UserContext: paCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			kafkaSourcesGVR: test.AllowAll,
		},
	}, {
		Name:        "edit user",
		UserContext: editCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			kafkaSourcesGVR: test.AllowAll,
		},
	}, {
		Name:        "view user",
		UserContext: viewCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			kafkaSourcesGVR: test.AllowViewOnly,
		},
	}}

	test.RunUserPermissionTests(t, objects, tests...)
}
