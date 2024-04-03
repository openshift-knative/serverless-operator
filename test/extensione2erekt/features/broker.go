package features

import (
	cetest "github.com/cloudevents/sdk-go/v2/test"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"knative.dev/eventing-kafka-broker/control-plane/pkg/kafka"
	"knative.dev/eventing-kafka-broker/test/rekt/resources/configmap"
	brokerconfigmap "knative.dev/eventing-kafka-broker/test/rekt/resources/configmap/broker"
	duckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	"knative.dev/eventing/pkg/apis/eventing"
	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/eventing/test/rekt/resources/trigger"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/manifest"
	"knative.dev/reconciler-test/pkg/resources/service"
)

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
		trigger.WithRetry(3, &backoffPolicy, ptr.To("PT1S")),
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
