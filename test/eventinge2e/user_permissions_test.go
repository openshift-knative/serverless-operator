package eventinge2e

import (
	"testing"

	eventingmessagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	eventingflowsv1 "knative.dev/eventing/pkg/apis/flows/v1"
	eventingsourcesv1beta2 "knative.dev/eventing/pkg/apis/sources/v1beta2"
)

func init() {
	eventingv1.AddToScheme(scheme.Scheme)
	eventingsourcesv1beta2.AddToScheme(scheme.Scheme)
	eventingmessagingv1.AddToScheme(scheme.Scheme)
	eventingflowsv1.AddToScheme(scheme.Scheme)
}

func TestEventingUserPermissions(t *testing.T) {
	paCtx := test.SetupProjectAdmin(t)
	editCtx := test.SetupEdit(t)
	viewCtx := test.SetupView(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, paCtx, editCtx, viewCtx) })
	defer test.CleanupAll(t, paCtx, editCtx, viewCtx)

	brokersGVR := eventingv1.SchemeGroupVersion.WithResource("brokers")
	pingSourcesGVR := eventingsourcesv1beta2.SchemeGroupVersion.WithResource("pingsources")
	channelsGVR := eventingmessagingv1.SchemeGroupVersion.WithResource("channels")
	sequencesGVR := eventingflowsv1.SchemeGroupVersion.WithResource("sequences")

	broker := &eventingv1.Broker{
		Spec: eventingv1.BrokerSpec{},
	}

	pingSource := &eventingsourcesv1beta2.PingSource{
		Spec: eventingsourcesv1beta2.PingSourceSpec{
			Data: "foo",
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

	imc := &eventingmessagingv1.Channel{}

	sequence := &eventingflowsv1.Sequence{
		Spec: eventingflowsv1.SequenceSpec{
			Steps: []eventingflowsv1.SequenceStep{
				{
					Destination: duckv1.Destination{
						URI: apis.HTTP("mydomain"),
					},
				},
			},
		},
	}

	objects := map[schema.GroupVersionResource]*unstructured.Unstructured{
		brokersGVR:     {},
		pingSourcesGVR: {},
		channelsGVR:    {},
		sequencesGVR:   {},
	}

	if err := scheme.Scheme.Convert(broker, objects[brokersGVR], nil); err != nil {
		t.Fatalf("Failed to convert Broker: %v", err)
	}
	if err := scheme.Scheme.Convert(pingSource, objects[pingSourcesGVR], nil); err != nil {
		t.Fatalf("Failed to convert PingSource: %v", err)
	}
	if err := scheme.Scheme.Convert(imc, objects[channelsGVR], nil); err != nil {
		t.Fatalf("Failed to convert Channel: %v", err)
	}
	if err := scheme.Scheme.Convert(sequence, objects[sequencesGVR], nil); err != nil {
		t.Fatalf("Failed to convert Sequence: %v", err)
	}

	tests := []test.UserPermissionTest{{
		Name:        "project admin user",
		UserContext: paCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			brokersGVR:     test.AllowAll,
			pingSourcesGVR: test.AllowAll,
			channelsGVR:    test.AllowAll,
			sequencesGVR:   test.AllowAll,
		},
	}, {
		Name:        "edit user",
		UserContext: editCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			brokersGVR:     test.AllowAll,
			pingSourcesGVR: test.AllowAll,
			channelsGVR:    test.AllowAll,
			sequencesGVR:   test.AllowAll,
		},
	}, {
		Name:        "view user",
		UserContext: viewCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			brokersGVR:     test.AllowViewOnly,
			pingSourcesGVR: test.AllowViewOnly,
			channelsGVR:    test.AllowViewOnly,
			sequencesGVR:   test.AllowViewOnly,
		},
	}}

	test.RunUserPermissionTests(t, objects, tests...)
}
