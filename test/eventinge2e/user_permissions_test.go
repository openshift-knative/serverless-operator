package eventinge2e

import (
	"context"
	"testing"

	eventingmessagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	eventingflowsv1 "knative.dev/eventing/pkg/apis/flows/v1"
	eventingsourcesv1beta2 "knative.dev/eventing/pkg/apis/sources/v1beta2"
)

type allowedOperations struct {
	get    bool
	list   bool
	create bool
	delete bool
}

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
			brokersGVR:     allowAll,
			pingSourcesGVR: allowAll,
			channelsGVR:    allowAll,
			sequencesGVR:   allowAll,
		},
	}, {
		name:        "edit user",
		userContext: editCtx,
		allowed: map[schema.GroupVersionResource]allowedOperations{
			brokersGVR:     allowAll,
			pingSourcesGVR: allowAll,
			channelsGVR:    allowAll,
			sequencesGVR:   allowAll,
		},
	}, {
		name:        "view user",
		userContext: viewCtx,
		allowed: map[schema.GroupVersionResource]allowedOperations{
			brokersGVR:     allowViewOnly,
			pingSourcesGVR: allowViewOnly,
			channelsGVR:    allowViewOnly,
			sequencesGVR:   allowViewOnly,
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for gvr, allowed := range tt.allowed {
				client := tt.userContext.Clients.Dynamic.Resource(gvr).Namespace(testNamespace)

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

				if err != nil {
					// If we've been able to delete the object we can assume we're able to get it as well.
					// Some objects take a while to be deleted, so we retry a few times.
					if err := wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
						_, err = client.Get(context.Background(), obj.GetName(), metav1.GetOptions{})
						if apierrs.IsNotFound(err) {
							return true, nil
						}
						return false, err
					}); err != nil {
						t.Fatalf("Unexpected error waiting for %s to be deleted, err = %v", gvr.String(), err)
					}
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
