package extensione2erekt

import (
	"testing"

	"knative.dev/eventing/test/rekt/features/channel"
	"knative.dev/eventing/test/rekt/resources/subscription"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/reconciler-test/pkg/manifest"
)

// ContainerSource -> KafkaChannel -> Subscription -> Ksvc -> Sink (Eventshub)
func TestContainerSourceKafkaChannelKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	createSubscriberFn := func(ref *duckv1.KReference, uri string) manifest.CfgFn {
		return subscription.WithSubscriber(ref, uri)
	}

	env.Test(ctx, t, channel.ChannelChain(1, createSubscriberFn))
}
