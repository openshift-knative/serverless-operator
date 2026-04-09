package extensione2erekt

import (
	"context"
	"testing"
	"time"

	"knative.dev/eventing/test/rekt/features/channel"
	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/eventing/test/rekt/resources/subscription"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/manifest"

	kafkafeatures "github.com/openshift-knative/serverless-operator/test/extensione2erekt/features"
)

// ContainerSource -> KafkaChannel -> Subscription -> Ksvc -> Sink (Eventshub)
func TestContainerSourceKafkaChannelKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	createSubscriberFn := func(ref *duckv1.KReference, uri string) manifest.CfgFn {
		return subscription.WithSubscriber(ref, uri, "")
	}

	since := time.Now()

	env.Test(ctx, t, kafkaChannelChain(1, createSubscriberFn))

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, kafkafeatures.VerifyEncryptedTrafficForKafkaChannel(since))
	}
}

// kafkaChannelChain wraps the upstream ChannelChain with an additional wait step
// to ensure the KafkaChannel dispatcher contract has been propagated before the
// ContainerSource starts sending events. Without this, there is a race condition
// where the dispatcher's Kafka consumer joins the consumer group but has no egress
// configured yet, causing consumed events to be silently dropped.
func kafkaChannelChain(length int, createSubscriberFn func(ref *duckv1.KReference, uri string) manifest.CfgFn) *feature.Feature {
	f := feature.NewFeature()

	sink, ch := channel.ChannelChainSetup(f, length, createSubscriberFn)

	// Wait for the channel to be addressable and give the kafka-controller time
	// to update the dispatcher's contract ConfigMap with the egress entry.
	f.Requirement("channel is addressable", channel_impl.IsAddressable(ch))
	f.Requirement("wait for dispatcher contract propagation", func(ctx context.Context, t feature.T) {
		time.Sleep(10 * time.Second)
	})

	channel.ChannelChainAssert(f, sink, ch)

	return f
}
