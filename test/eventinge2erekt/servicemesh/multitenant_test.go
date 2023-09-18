package servicemesh

import (
	"context"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/test"
	"knative.dev/eventing/test/rekt/features"
	"knative.dev/eventing/test/rekt/resources/pingsource"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
	"knative.dev/reconciler-test/pkg/resources/service"
)

// PingSource -> Ksvc -> Sink (Eventshub)
func TestPingSourceToKsvc(t *testing.T) {
	t.Parallel()

	ctxTenant1, envTenant1 := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		eventshub.WithKnativeServiceForwarder,
		environment.WithPollTimings(5*time.Second, 4*time.Minute),
		environment.WithNamespace("tenant-1"),
		environment.Managed(t),
	)

	ctxTenant2, envTenant2 := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		eventshub.WithKnativeServiceForwarder,
		environment.WithPollTimings(5*time.Second, 1*time.Minute),
		environment.WithNamespace("tenant-2"),
		environment.Managed(t),
	)

	sink := feature.MakeRandomK8sName("sink")

	// Deploy sink in tenant-1.
	envTenant1.Test(ctxTenant1, t, KsvcSink(sink))
	// Check PingSource deployed in tenant-2
	envTenant2.Test(ctxTenant2, t, VerifyPingSource(sink, ctxTenant1))
}

func KsvcSink(name string) *feature.Feature {
	f := feature.NewFeature()

	f.Setup("install sink", eventshub.Install(name, eventshub.StartReceiver))

	return f
}

func VerifyPingSource(sink string, otherContext context.Context) *feature.Feature {
	source := feature.MakeRandomK8sName("pingsource")
	f := feature.NewFeature()

	sinkRef := service.AsKReference(sink)
	sinkRef.Namespace = environment.FromContext(otherContext).Namespace()
	f.Requirement("install pingsource", pingsource.Install(source, pingsource.WithSink(sinkRef, "")))
	f.Requirement("pingsource goes ready", pingsource.IsReady(source))

	f.Stable("pingsource as event source").
		Must("delivers events",
			func(ctx context.Context, t feature.T) {
				assert.OnStore(sink).
					Match(features.HasKnNamespaceHeader(environment.FromContext(ctx).Namespace())).
					MatchEvent(test.HasType("dev.knative.sources.ping")).
					AtLeast(1)(otherContext, t) // Use the other context for checking event store.
			},
		)

	return f
}
