package servicemesh

import (
	"context"
	"testing"
	"time"

	cetest "github.com/cloudevents/sdk-go/v2/test"
	"github.com/google/uuid"
	kafkafeatures "github.com/openshift-knative/serverless-operator/test/extensione2erekt/features"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"knative.dev/eventing-kafka-broker/control-plane/pkg/kafka"
	"knative.dev/eventing-kafka-broker/test/rekt/resources/configmap"
	duckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	"knative.dev/eventing/test/rekt/features"
	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/eventing/test/rekt/resources/containersource"
	"knative.dev/eventing/test/rekt/resources/subscription"
	"knative.dev/eventing/test/rekt/resources/trigger"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
	"knative.dev/reconciler-test/pkg/manifest"
	"knative.dev/reconciler-test/pkg/resources/service"
)

// ContainerSource (tenant-2) -> KafkaChannel (tenant-1) -> Subscription -> Ksvc (tenant-1) -> Sink (tenant-1)
func TestContainerSourceKafkaChannelKsvcCrossTenant(t *testing.T) {
	t.Parallel()

	ctxTenant1, envTenant1 := environmentWithNamespace(t, "tenant-1")
	ctxTenant2, envTenant2 := environmentWithNamespace(t, "tenant-2")

	channel := feature.MakeRandomK8sName("channel")
	sink := feature.MakeRandomK8sName("sink")

	// Deploy sink in tenant-1.
	envTenant1.Test(ctxTenant1, t, DeployKafkaChannelKsvc(channel, sink))
	// Check cross-tenant event.
	envTenant2.Test(ctxTenant2, t, VerifyContainerSourceToChannelBlocked(channel, sink, ctxTenant1, time.Now()))
}

func DeployKafkaChannelKsvc(channel, sink string) *feature.Feature {
	f := feature.NewFeature()

	f.Setup("install channel", channel_impl.Install(channel))
	f.Setup("channel is ready", channel_impl.IsReady(channel))

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver))

	sub := feature.MakeRandomK8sName("subscription")
	f.Setup("install sink subscription", subscription.Install(sub,
		subscription.WithChannel(channel_impl.AsRef(channel)),
		subscription.WithSubscriber(service.AsKReference(sink), ""),
	))

	f.Setup("subscription is ready", subscription.IsReady(sub))

	return f
}

func VerifyContainerSourceToChannelBlocked(channel, sink string, channelCtx context.Context, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	cs := feature.MakeRandomK8sName("containersource")

	channelRef := channel_impl.AsRef(channel)
	channelRef.Namespace = environment.FromContext(channelCtx).Namespace()

	f.Setup("install containersource", containersource.Install(cs, containersource.WithSink(channelRef, "")))
	f.Setup("containersource goes ready", containersource.IsReady(cs))

	f.Assert("container source does not deliver event to channel across tenants",
		func(ctx context.Context, t feature.T) {
			assert.OnStore(sink).
				MatchEvent(cetest.HasType("dev.knative.eventing.samples.heartbeat")).
				Not()(channelCtx, t)
		},
	)

	f.Assert("request to kafka channel is forbidden", func(ctx context.Context, t feature.T) {
		kafkafeatures.VerifyEncryptedTrafficToKafkaChannel(
			environment.FromContext(channelCtx).References(), since, true /*trafficBlocked*/)
	})

	return f
}

// ContainerSource (tenant-2) -> KafkaChannel (tenant-2) -> Subscription -> Sink (tenant-2)
//
//	-> subscription with reply to Sink (tenant-1)
//	-> subscription with deadLetterSink to Sink (tenant-2)
//
// The sink in "reply" should not receive any event because it's in a different tenant.
// The original sink and dead letter sink should receive an event.
func TestContainerSourceKafkaChannelKsvcWithReplyAndDLSCrossTenant(t *testing.T) {
	t.Parallel()

	// Do not use Knative Service Forwarder as it doesn't work correctly with replies.
	ctxTenant1, envTenant1 := global.Environment(
		environment.WithNamespace("tenant-1"),
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(5*time.Second, 4*time.Minute),
		environment.Managed(t),
	)
	ctxTenant2, envTenant2 := global.Environment(
		environment.WithNamespace("tenant-2"),
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(5*time.Second, 4*time.Minute),
		environment.Managed(t),
	)

	replySink := feature.MakeRandomK8sName("sink")

	// Deploy reply-sink in tenant-1.
	envTenant1.Test(ctxTenant1, t, DeploySink(replySink))
	// Check cross-tenant event.
	envTenant2.Test(ctxTenant2, t, VerifyContainerSourceToChannelWithReplyAndDLS(replySink, ctxTenant1, time.Now()))
}

func DeploySink(sink string) *feature.Feature {
	f := feature.NewFeature()

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver))

	return f
}

func VerifyContainerSourceToChannelWithReplyAndDLS(replySink string, replySinkCtx context.Context, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	channel := feature.MakeRandomK8sName("channel")
	sink := feature.MakeRandomK8sName("sink")
	dls := feature.MakeRandomK8sName("dls")
	cs := feature.MakeRandomK8sName("containersource")

	f.Setup("install channel", channel_impl.Install(channel))
	f.Setup("channel is ready", channel_impl.IsReady(channel))

	f.Setup("install sink", eventshub.Install(sink,
		eventshub.StartReceiver, eventshub.EchoEvent))
	f.Setup("install dls", eventshub.Install(dls, eventshub.StartReceiver))

	replySinkRef := service.AsKReference(replySink)
	replySinkRef.Namespace = environment.FromContext(replySinkCtx).Namespace()

	sub := feature.MakeRandomK8sName("subscription")
	f.Setup("install sink subscription", subscription.Install(sub,
		subscription.WithChannel(channel_impl.AsRef(channel)),
		subscription.WithSubscriber(service.AsKReference(sink), ""),
		subscription.WithReply(replySinkRef, ""),
		subscription.WithDeadLetterSink(service.AsKReference(dls), ""),
	))

	f.Setup("subscription is ready", subscription.IsReady(sub))

	f.Requirement("install containersource", containersource.Install(cs,
		containersource.WithSink(channel_impl.AsRef(channel), "")))
	f.Requirement("containersource goes ready", containersource.IsReady(cs))

	f.Assert("sink receives events", func(ctx context.Context, t feature.T) {
		assert.OnStore(sink).
			Match(features.HasKnNamespaceHeader(environment.FromContext(ctx).Namespace())).
			MatchEvent(cetest.HasType("dev.knative.eventing.samples.heartbeat")).
			AtLeast(1)(ctx, t)
	})

	f.Assert("deadlettersink receives events", func(ctx context.Context, t feature.T) {
		assert.OnStore(dls).
			Match(features.HasKnNamespaceHeader(environment.FromContext(ctx).Namespace())).
			MatchEvent(cetest.HasType("dev.knative.eventing.samples.heartbeat")).
			AtLeast(1)(ctx, t)
	})

	f.Assert("reply sink does not receive events", func(ctx context.Context, t feature.T) {
		assert.OnStore(replySink).
			Match(features.HasKnNamespaceHeader(environment.FromContext(ctx).Namespace())).
			MatchEvent(cetest.HasType("dev.knative.eventing.samples.heartbeat")).
			Not()(replySinkCtx, t)
	})

	f.Assert("request to event sink is forbidden", func(ctx context.Context, t feature.T) {
		kafkafeatures.VerifyRequestToSinkForbidden(replySink, environment.FromContext(replySinkCtx).Namespace(), since)
	})

	return f
}

// Source (Eventshub) -> KafkaBroker -> Trigger -> Ksvc -> Sink (Eventshub)
func TestSourceToKafkaBrokerKsvcCrossTenant(t *testing.T) {
	t.Parallel()

	ctxTenant1, envTenant1 := environmentWithNamespace(t, "tenant-1")
	ctxTenant2, envTenant2 := environmentWithNamespace(t, "tenant-2")

	broker := feature.MakeRandomK8sName("broker")
	sink := feature.MakeRandomK8sName("sink")

	// Deploy sink in tenant-1.
	envTenant1.Test(ctxTenant1, t, DeployBrokerTriggerKsvc(broker, sink))
	// Check cross-tenant event.
	envTenant2.Test(ctxTenant2, t, VerifySourceToKafkaBrokerBlocked(broker, sink, ctxTenant1, time.Now()))
}

func DeployBrokerTriggerKsvc(brokerName, sink string) *feature.Feature {
	f := feature.NewFeatureNamed("broker smoke test")

	config := feature.MakeRandomK8sName("kafka-broker-config")
	triggerName := feature.MakeRandomK8sName("trigger")

	f.Setup("create broker config", configmap.Copy(
		types.NamespacedName{Namespace: system.Namespace(), Name: "kafka-broker-config"},
		config,
	))

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver))

	f.Setup("install broker", broker.Install(brokerName,
		append([]manifest.CfgFn{broker.WithConfig(config)}, broker.WithBrokerClass(kafka.BrokerClass))...))
	f.Setup("broker ready", broker.IsReady(brokerName))

	backoffPolicy := duckv1.BackoffPolicyLinear
	f.Requirement("install trigger", trigger.Install(
		triggerName,
		brokerName,
		trigger.WithRetry(3, &backoffPolicy, pointer.String("PT1S")),
		trigger.WithSubscriber(service.AsKReference(sink), ""),
	))
	f.Requirement("trigger ready", trigger.IsReady(triggerName))

	return f
}

func VerifySourceToKafkaBrokerBlocked(brokerName, sink string, sinkCtx context.Context, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	event := cetest.FullEvent()
	event.SetID(uuid.New().String())

	eventMatchers := []cetest.EventMatcher{
		cetest.HasId(event.ID()),
		cetest.HasSource(event.Source()),
		cetest.HasType(event.Type()),
		cetest.HasSubject(event.Subject()),
	}

	f.Requirement("install eventshub source",
		func(ctx context.Context, t feature.T) {
			u, err := k8s.Address(sinkCtx, broker.GVR(), brokerName)
			if err != nil {
				t.Fatal(err)
			}
			eventshub.Install(
				feature.MakeRandomK8sName("source"),
				eventshub.StartSenderURL(u.String()),
				eventshub.InputEvent(event),
			)(ctx, t)
		},
	)

	f.Assert("source does not deliver event to kafka broker across tenants",
		func(ctx context.Context, t feature.T) {
			assert.OnStore(sink).
				MatchEvent(eventMatchers...).
				Not()(sinkCtx, t)
		},
	)

	f.Assert("request to kafka broker is forbidden", func(ctx context.Context, t feature.T) {
		kafkafeatures.VerifyEncryptedTrafficToKafkaBroker(
			environment.FromContext(sinkCtx).References(), false /*namespaced*/, since, true /*trafficBlocked*/)
	})

	return f
}
