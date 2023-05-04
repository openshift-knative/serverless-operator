package eventinge2erekt

import (
	"testing"

	"knative.dev/eventing/test/rekt/features/broker"
	resources "knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

func TestChannelBasedBrokerToKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		// Enables KnativeService in the scenario.
		eventshub.WithKnativeServiceForwarder,
		environment.Managed(t),
	)

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		t.Skip("Channel-based tests cannot run in service mesh mode for now")
	}

	env.Prerequisite(ctx, t, broker.GoesReady("default", resources.WithEnvConfig()...))
	env.Test(ctx, t, broker.SourceToSink("default"))
}
