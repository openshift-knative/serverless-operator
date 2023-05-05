package eventinge2erekt

import (
	"testing"

	"knative.dev/eventing/test/rekt/features/channel"
	"knative.dev/eventing/test/rekt/resources/subscription"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
	"knative.dev/reconciler-test/pkg/manifest"
)

// ContainerSource -> Channel -> Subscription -> Ksvc -> Sink (Eventshub)
func TestContainerSourceChannelKsvc(t *testing.T) {
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

	createSubscriberFn := func(ref *duckv1.KReference, uri string) manifest.CfgFn {
		return subscription.WithSubscriber(ref, uri)
	}
	env.Test(ctx, t, channel.ChannelChain(1, createSubscriberFn))
}
