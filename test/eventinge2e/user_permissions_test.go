package eventinge2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	eventingv1beta1 "knative.dev/eventing/pkg/apis/eventing/v1beta1"
	flowsv1 "knative.dev/eventing/pkg/apis/flows/v1"
	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	sourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/tracker"
)

func init() {
	eventingv1.AddToScheme(scheme.Scheme)
	eventingv1beta1.AddToScheme(scheme.Scheme)
	sourcesv1.AddToScheme(scheme.Scheme)
	messagingv1.AddToScheme(scheme.Scheme)
	flowsv1.AddToScheme(scheme.Scheme)
}

const (
	ksvcAPIVersion    = "serving.knative.dev/v1"
	ksvcKind          = "Service"
	channelAPIVersion = "messaging.knative.dev/v1"
	channelKind       = "Channel"
)

func TestEventingUserPermissions(t *testing.T) {
	paCtx := test.SetupProjectAdmin(t)
	editCtx := test.SetupEdit(t)
	viewCtx := test.SetupView(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, paCtx, editCtx, viewCtx) })
	defer test.CleanupAll(t, paCtx, editCtx, viewCtx)

	brokersGVR := eventingv1.SchemeGroupVersion.WithResource("brokers")
	pingSourcesGVR := sourcesv1.SchemeGroupVersion.WithResource("pingsources")
	channelsGVR := messagingv1.SchemeGroupVersion.WithResource("channels")
	sequencesGVR := flowsv1.SchemeGroupVersion.WithResource("sequences")
	apiServerSourcesGVR := sourcesv1.SchemeGroupVersion.WithResource("apiserversources")
	containerSourcesGVR := sourcesv1.SchemeGroupVersion.WithResource("containersources")
	eventTypesGVR := eventingv1beta1.SchemeGroupVersion.WithResource("eventtypes")
	inMemoryChannelsGCR := messagingv1.SchemeGroupVersion.WithResource("inmemorychannels")
	parallelsGVR := flowsv1.SchemeGroupVersion.WithResource("parallels")
	sinkBindingsGVR := sourcesv1.SchemeGroupVersion.WithResource("sinkbindings")
	subscriptionsGVR := messagingv1.SchemeGroupVersion.WithResource("subscriptions")

	broker := &eventingv1.Broker{
		Spec: eventingv1.BrokerSpec{},
	}

	pingSource := &sourcesv1.PingSource{
		Spec: sourcesv1.PingSourceSpec{
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

	imc := &messagingv1.Channel{}

	sequence := &flowsv1.Sequence{
		Spec: flowsv1.SequenceSpec{
			Steps: []flowsv1.SequenceStep{
				{
					Destination: duckv1.Destination{
						URI: apis.HTTP("mydomain"),
					},
				},
			},
		},
	}

	apiServerSource := &sourcesv1.ApiServerSource{
		Spec: sourcesv1.ApiServerSourceSpec{
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: ksvcAPIVersion,
						Kind:       ksvcKind,
						Name:       "fakeKSVC",
					},
				},
			},
			Resources: []sourcesv1.APIVersionKindSelector{
				{
					APIVersion: "v1",
					Kind:       "Event",
				},
			},
		},
	}

	containerSource := &sourcesv1.ContainerSource{
		Spec: sourcesv1.ContainerSourceSpec{
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: ksvcAPIVersion,
						Kind:       ksvcKind,
						Name:       "fakeKSVC",
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-container",
							Image: "my-image",
						},
					},
				},
			},
		},
	}

	eventType := &eventingv1beta1.EventType{
		Spec: eventingv1beta1.EventTypeSpec{
			Type: "dev.knative.source.github.push",
		},
	}

	inMemoryChannel := &messagingv1.InMemoryChannel{
		Spec: messagingv1.InMemoryChannelSpec{},
	}

	parallel := &flowsv1.Parallel{
		Spec: flowsv1.ParallelSpec{
			Branches: []flowsv1.ParallelBranch{
				{
					Subscriber: duckv1.Destination{
						Ref: &duckv1.KReference{
							APIVersion: ksvcAPIVersion,
							Kind:       ksvcKind,
							Name:       "fakeKSVC",
						},
					},
				},
			},
		},
	}

	sinkBinding := &sourcesv1.SinkBinding{
		Spec: sourcesv1.SinkBindingSpec{
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: ksvcAPIVersion,
						Kind:       ksvcKind,
						Name:       "fakeKSVC",
					},
				},
			},
			BindingSpec: duckv1.BindingSpec{
				Subject: tracker.Reference{
					Name:       "my-job",
					APIVersion: "batch/v1",
					Kind:       "Job",
					Namespace:  test.Namespace,
				},
			},
		},
	}

	subscription := &messagingv1.Subscription{
		Spec: messagingv1.SubscriptionSpec{
			Channel: duckv1.KReference{
				APIVersion: channelAPIVersion,
				Kind:       channelKind,
				Name:       "channel",
			},
			Subscriber: &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: ksvcAPIVersion,
					Kind:       ksvcKind,
					Name:       "fakeKSVC",
				},
			},
		},
	}

	objects := map[schema.GroupVersionResource]*unstructured.Unstructured{
		brokersGVR:          {},
		pingSourcesGVR:      {},
		channelsGVR:         {},
		sequencesGVR:        {},
		apiServerSourcesGVR: {},
		containerSourcesGVR: {},
		eventTypesGVR:       {},
		inMemoryChannelsGCR: {},
		parallelsGVR:        {},
		sinkBindingsGVR:     {},
		subscriptionsGVR:    {},
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
	if err := scheme.Scheme.Convert(apiServerSource, objects[apiServerSourcesGVR], nil); err != nil {
		t.Fatalf("Failed to convert ApiServerSource: %v", err)
	}
	if err := scheme.Scheme.Convert(containerSource, objects[containerSourcesGVR], nil); err != nil {
		t.Fatalf("Failed to convert ContainerSource: %v", err)
	}
	if err := scheme.Scheme.Convert(eventType, objects[eventTypesGVR], nil); err != nil {
		t.Fatalf("Failed to convert EventType: %v", err)
	}
	if err := scheme.Scheme.Convert(inMemoryChannel, objects[inMemoryChannelsGCR], nil); err != nil {
		t.Fatalf("Failed to convert InMemoryChannel: %v", err)
	}
	if err := scheme.Scheme.Convert(parallel, objects[parallelsGVR], nil); err != nil {
		t.Fatalf("Failed to convert Parallel: %v", err)
	}
	if err := scheme.Scheme.Convert(sinkBinding, objects[sinkBindingsGVR], nil); err != nil {
		t.Fatalf("Failed to convert SinkBinding: %v", err)
	}
	if err := scheme.Scheme.Convert(subscription, objects[subscriptionsGVR], nil); err != nil {
		t.Fatalf("Failed to convert Subscription: %v", err)
	}

	tests := []test.UserPermissionTest{{
		Name:        "project admin user",
		UserContext: paCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			brokersGVR:          test.AllowAll,
			pingSourcesGVR:      test.AllowAll,
			channelsGVR:         test.AllowAll,
			sequencesGVR:        test.AllowAll,
			apiServerSourcesGVR: test.AllowAll,
			containerSourcesGVR: test.AllowAll,
			eventTypesGVR:       test.AllowAll,
			inMemoryChannelsGCR: test.AllowAll,
			parallelsGVR:        test.AllowAll,
			sinkBindingsGVR:     test.AllowAll,
			subscriptionsGVR:    test.AllowAll,
		},
	}, {
		Name:        "edit user",
		UserContext: editCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			brokersGVR:          test.AllowAll,
			pingSourcesGVR:      test.AllowAll,
			channelsGVR:         test.AllowAll,
			sequencesGVR:        test.AllowAll,
			apiServerSourcesGVR: test.AllowAll,
			containerSourcesGVR: test.AllowAll,
			eventTypesGVR:       test.AllowAll,
			inMemoryChannelsGCR: test.AllowAll,
			parallelsGVR:        test.AllowAll,
			sinkBindingsGVR:     test.AllowAll,
			subscriptionsGVR:    test.AllowAll,
		},
	}, {
		Name:        "view user",
		UserContext: viewCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			brokersGVR:          test.AllowViewOnly,
			pingSourcesGVR:      test.AllowViewOnly,
			channelsGVR:         test.AllowViewOnly,
			sequencesGVR:        test.AllowViewOnly,
			apiServerSourcesGVR: test.AllowViewOnly,
			containerSourcesGVR: test.AllowViewOnly,
			eventTypesGVR:       test.AllowViewOnly,
			inMemoryChannelsGCR: test.AllowViewOnly,
			parallelsGVR:        test.AllowViewOnly,
			sinkBindingsGVR:     test.AllowViewOnly,
			subscriptionsGVR:    test.AllowViewOnly,
		},
	}}

	test.RunUserPermissionTests(t, objects, tests...)
}
