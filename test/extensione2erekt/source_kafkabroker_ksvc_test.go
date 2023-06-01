package extensione2erekt

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
	brokerconfigmap "knative.dev/eventing-kafka-broker/test/rekt/resources/configmap/broker"
	duckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	"knative.dev/eventing/pkg/apis/eventing"
	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/eventing/test/rekt/resources/trigger"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/manifest"
	"knative.dev/reconciler-test/pkg/resources/service"

	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
)

// Source (Eventshub) -> KafkaBroker -> Trigger -> Ksvc -> Sink (Eventshub)
func TestSourceKafkaBrokerKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	since := time.Now()

	env.Test(ctx, t, BrokerSmokeTest(kafka.BrokerClass))
	env.Test(ctx, t, VerifyMetricsKafkaBroker())

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, kafkafeatures.VerifyEncryptedTrafficForKafkaBroker(env.References(), since))
	}
}

// Source (Eventshub) -> NamespacedKafkaBroker -> Trigger -> Ksvc -> Sink (Eventshub)
func TestSourceNamespacedKafkaBrokerKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	since := time.Now()

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		// With Istio this issue happens often.
		t.Skip("https://issues.redhat.com/browse/SRVKE-1424")
	}

	env.Test(ctx, t, BrokerSmokeTest(kafka.NamespacedBrokerClass))
	env.Test(ctx, t, VerifyMetricsNamespacedKafkaBroker(environment.FromContext(ctx).Namespace()))

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, kafkafeatures.VerifyEncryptedTrafficForNamespacedKafkaBroker(env.References(), since))
	}
}

// Source (Eventshub) -> MTChannelBased-KafkaBroker -> Trigger -> Ksvc -> Sink (Eventshub)
func TestSourceChannelBasedKafkaBrokerKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		t.Skip("Channel-based tests cannot run in service mesh mode for now")
	}

	env.Test(ctx, t, BrokerSmokeTest(eventing.MTChannelBrokerClassValue))
}

func BrokerSmokeTest(brokerClass string) *feature.Feature {
	f := feature.NewFeatureNamed("broker smoke test")

	sink := feature.MakeRandomK8sName("sink")
	configName := feature.MakeRandomK8sName("kafka-broker-config")
	brokerName := feature.MakeRandomK8sName("broker")
	triggerName := feature.MakeRandomK8sName("trigger")

	switch brokerClass {
	// Both Kafka and NamespacedKafka support using ConfigMap from test namespace.
	case kafka.NamespacedBrokerClass, kafka.BrokerClass:
		f.Setup("create broker config", configmap.Copy(
			types.NamespacedName{Namespace: system.Namespace(), Name: "kafka-broker-config"},
			configName,
		))
	case eventing.MTChannelBrokerClassValue:
		opts := []manifest.CfgFn{brokerconfigmap.WithKafkaChannelMTBroker()}
		f.Setup("create broker config", brokerconfigmap.Install(configName, opts...))
	}

	event := cetest.FullEvent()
	event.SetID(uuid.New().String())

	eventMatchers := []cetest.EventMatcher{
		cetest.HasId(event.ID()),
		cetest.HasSource(event.Source()),
		cetest.HasType(event.Type()),
		cetest.HasSubject(event.Subject()),
	}

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver))

	f.Setup("install broker", broker.Install(brokerName,
		append([]manifest.CfgFn{broker.WithConfig(configName)}, broker.WithBrokerClass(brokerClass))...))
	f.Setup("broker ready", broker.IsReady(brokerName))

	backoffPolicy := duckv1.BackoffPolicyLinear
	f.Setup("install trigger", trigger.Install(
		triggerName,
		brokerName,
		trigger.WithRetry(3, &backoffPolicy, pointer.String("PT1S")),
		trigger.WithSubscriber(service.AsKReference(sink), ""),
	))
	f.Setup("trigger ready", trigger.IsReady(triggerName))

	f.Requirement("install eventshub source", eventshub.Install(
		feature.MakeRandomK8sName("source"),
		eventshub.StartSenderToResource(broker.GVR(), brokerName),
		eventshub.InputEvent(event),
	))

	f.Assert("sink receives event", assert.OnStore(sink).MatchEvent(eventMatchers...).Exact(1))

	return f
}

func VerifyMetricsKafkaBroker() *feature.Feature {
	f := feature.NewFeature()

	f.Stable("kafka broker").
		Must("has metrics", func(ctx context.Context, t feature.T) {
			if err := monitoringe2e.VerifyMetrics(ctx, monitoringe2e.KafkaBrokerDataPlaneQueries); err != nil {
				t.Fatal("Failed to verify that Kafka Broker data plane metrics work correctly", err)
			}
		})

	return f
}

func VerifyMetricsNamespacedKafkaBroker(namespace string) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("namespaced kafka broker").
		Must("has metrics", func(ctx context.Context, t feature.T) {
			if err := monitoringe2e.VerifyMetrics(ctx, monitoringe2e.NamespacedKafkaBrokerDataPlaneQueries(namespace)); err != nil {
				t.Fatal("Failed to verify that Kafka Broker data plane metrics work correctly", err)
			}
		})

	return f
}
