package features

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/resources/knativeservice"
	"knative.dev/reconciler-test/pkg/resources/service"

	"knative.dev/eventing/test/rekt/features"
	"knative.dev/eventing/test/rekt/resources/pingsource"
)

// PingSourceSendsEventsWithSinkRef is a downstream variant of
// pingsource.SendsEventsWithSinkRef that prevents the eventshub forwarder
// Knative Service from scaling to zero. With Service Mesh, the Envoy sidecar
// may fail to reroute traffic to the activator after scale-to-zero, causing
// event delivery to hang silently.
func PingSourceSendsEventsWithSinkRef() *feature.Feature {
	source := feature.MakeRandomK8sName("pingsource")
	sink := feature.MakeRandomK8sName("sink")
	f := feature.NewFeature()

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver))

	f.Requirement("set min-scale on forwarder", setMinScale(sink, "1"))
	f.Requirement("install pingsource", pingsource.Install(source, pingsource.WithSink(service.AsDestinationRef(sink))))
	f.Requirement("pingsource goes ready", pingsource.IsReady(source))

	f.Stable("pingsource as event source").
		Must("delivers events",
			func(ctx context.Context, t feature.T) {
				assert.OnStore(sink).
					Match(features.HasKnNamespaceHeader(environment.FromContext(ctx).Namespace())).
					MatchEvent(test.HasType("dev.knative.sources.ping")).
					AtLeast(1)(ctx, t)
			},
		)

	return f
}

// setMinScale patches a Knative Service's revision template with the
// autoscaling.knative.dev/min-scale annotation to prevent scale-to-zero.
func setMinScale(name, minScale string) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		namespace := environment.FromContext(ctx).Namespace()
		patch := fmt.Sprintf(
			`{"spec":{"template":{"metadata":{"annotations":{"autoscaling.knative.dev/min-scale":"%s"}}}}}`,
			minScale,
		)
		_, err := dynamicclient.Get(ctx).
			Resource(knativeservice.GVR()).
			Namespace(namespace).
			Patch(ctx, name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
		if err != nil {
			t.Fatalf("Failed to set min-scale on Knative Service %s/%s: %v", namespace, name, err)
		}
	}
}
