package knativekafkae2e

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	kafkasinksv1alpha1 "knative.dev/eventing-kafka-broker/control-plane/pkg/apis/eventing/v1alpha1"
	kafkabindingv1beta1 "knative.dev/eventing-kafka/pkg/apis/bindings/v1beta1"
	kafkachannelv1beta1 "knative.dev/eventing-kafka/pkg/apis/messaging/v1beta1"
	kafkasourcesv1beta1 "knative.dev/eventing-kafka/pkg/apis/sources/v1beta1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/ptr"

	"github.com/openshift-knative/serverless-operator/test"
)

func init() {
	kafkasourcesv1beta1.AddToScheme(scheme.Scheme)
	kafkachannelv1beta1.AddToScheme(scheme.Scheme)
	kafkabindingv1beta1.AddToScheme(scheme.Scheme)
	kafkasinksv1alpha1.AddToScheme(scheme.Scheme)
}

func TestKafkaUserPermissions(t *testing.T) {
	t.Skip()

	paCtx := test.SetupProjectAdmin(t)
	editCtx := test.SetupEdit(t)
	viewCtx := test.SetupView(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, paCtx, editCtx, viewCtx) })
	defer test.CleanupAll(t, paCtx, editCtx, viewCtx)

	kafkaSourcesGVR := kafkasourcesv1beta1.SchemeGroupVersion.WithResource("kafkasources")
	kafkaChannelsGVR := kafkachannelv1beta1.SchemeGroupVersion.WithResource("kafkachannels")
	kafkaSinksGVR := kafkasinksv1alpha1.SchemeGroupVersion.WithResource("kafkasinks")

	kafkaSource := &kafkasourcesv1beta1.KafkaSource{
		Spec: kafkasourcesv1beta1.KafkaSourceSpec{
			KafkaAuthSpec: kafkabindingv1beta1.KafkaAuthSpec{
				BootstrapServers: []string{plainBootstrapServer},
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

	kafkaChannel := &kafkachannelv1beta1.KafkaChannel{
		Spec: kafkachannelv1beta1.KafkaChannelSpec{
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	kafkaSink := &kafkasinksv1alpha1.KafkaSink{
		Spec: kafkasinksv1alpha1.KafkaSinkSpec{
			Topic:             "my-topic",
			NumPartitions:     ptr.Int32(10),
			ReplicationFactor: func(rf int16) *int16 { return &rf }(1),
			BootstrapServers:  []string{plainBootstrapServer},
			ContentMode:       ptr.String(kafkasinksv1alpha1.ModeStructured),
		},
	}

	objects := map[schema.GroupVersionResource]*unstructured.Unstructured{
		kafkaSourcesGVR:  {},
		kafkaChannelsGVR: {},
		kafkaSinksGVR:    {},
	}

	if err := scheme.Scheme.Convert(kafkaSource, objects[kafkaSourcesGVR], nil); err != nil {
		t.Fatalf("Failed to convert KafkaSource: %v", err)
	}
	if err := scheme.Scheme.Convert(kafkaChannel, objects[kafkaChannelsGVR], nil); err != nil {
		t.Fatalf("Failed to convert KafkaChannel: %v", err)
	}
	if err := scheme.Scheme.Convert(kafkaSink, objects[kafkaSinksGVR], nil); err != nil {
		t.Fatalf("Failed to convert KafkaSink: %v", err)
	}

	tests := []test.UserPermissionTest{{
		Name:        "project admin user",
		UserContext: paCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			kafkaSourcesGVR:  test.AllowAll,
			kafkaChannelsGVR: test.AllowAll,
			kafkaSinksGVR:    test.AllowAll,
		},
	}, {
		Name:        "edit user",
		UserContext: editCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			kafkaSourcesGVR:  test.AllowAll,
			kafkaChannelsGVR: test.AllowAll,
			kafkaSinksGVR:    test.AllowAll,
		},
	}, {
		Name:        "view user",
		UserContext: viewCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			kafkaSourcesGVR:  test.AllowViewOnly,
			kafkaChannelsGVR: test.AllowViewOnly,
			kafkaSinksGVR:    test.AllowViewOnly,
		},
	}}

	test.RunUserPermissionTests(t, objects, tests...)
}
