package extensione2erekt

import (
	"testing"
	"time"

	"knative.dev/eventing/test/rekt/features/channel"
	"knative.dev/eventing/test/rekt/resources/subscription"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/reconciler-test/pkg/environment"
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

	env.Test(ctx, t, channel.ChannelChain(1, createSubscriberFn))

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, kafkafeatures.VerifyEncryptedTrafficForKafkaChannel(since))
	}
}
