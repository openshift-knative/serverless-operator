package features

import (
	"fmt"

	"knative.dev/eventing/test/rekt/resources/delivery"
	triggerresources "knative.dev/eventing/test/rekt/resources/trigger"
	"knative.dev/reconciler-test/pkg/feature"
)

var sinksAll = []component{
	kafkaChannel,
	inMemoryChannel,
	genericChannelWithKafkaChannelTemplate,
	genericChannelWithInMemoryChannelTemplate,
	inMemoryChannelMtBroker,
	kafkaChannelMtBroker,
	kafkaBroker,
	inMemoryChannelSequence,
	kafkaChannelSequence,
	inMemoryChannelParallel,
	kafkaChannelParallel,
	ksvc,
}

var sinksShort = []component{
	kafkaChannel,
	genericChannelWithKafkaChannelTemplate,
	kafkaChannelMtBroker,
	kafkaBroker,
	kafkaChannelSequence,
	kafkaChannelParallel,
	ksvc,
}

var brokers = []component{
	inMemoryChannelMtBroker,
	kafkaChannelMtBroker,
	kafkaBroker,
}

var (
	deadLetterSinks      = sinksAll
	deadLetterSinksShort = sinksShort
	triggers             = sinksAll
)

func BrokerReadiness(broker component, brokerDls component, triggers []component, triggerDls component) *feature.Feature {
	testLabel := shortLabel(broker)
	if brokerDls != nil {
		testLabel = testLabel + "b" + shortLabel(brokerDls)
	}
	if triggerDls != nil {
		testLabel = testLabel + "t" + shortLabel(triggerDls)
	}

	brokerName := testLabel
	dlsName := testLabel + "-bdls"
	triggerDlsName := testLabel + "-tdls"
	f := feature.NewFeatureNamed(fmt.Sprintf("%s with %s Broker dls and %s Trigger dls", label(broker), label(brokerDls), label(triggerDls)))

	if triggerDls != nil {
		f.Setup("Install A Trigger DLS Thing", triggerDls.Install(triggerDlsName))
	}

	if brokerDls != nil {
		f.Setup("Install A Broker DLS Thing", brokerDls.Install(dlsName))
		f.Setup("Install A Broker with Delivery", broker.Install(brokerName, delivery.WithDeadLetterSink(brokerDls.KReference(dlsName), "")))
	} else {
		f.Setup("Install A Broker", broker.Install(brokerName))
	}

	for _, trigger := range triggers {
		// We'll label the Trigger based on the trigger component label, which should be unique
		triggerName := brokerName + "-" + trigger.ShortLabel()

		f.Setup(fmt.Sprintf("Install a %s Trigger Thing", trigger.Label()), trigger.Install(triggerName))

		if triggerDls != nil {
			f.Setup(fmt.Sprintf("Install a %s Trigger", trigger.Label()), triggerresources.Install(
				triggerName,
				brokerName,
				triggerresources.WithSubscriber(trigger.KReference(triggerName), ""),
				triggerresources.WithDeadLetterSink(triggerDls.KReference(triggerDlsName), "")))
		} else {
			f.Setup(fmt.Sprintf("Install a %s Trigger", trigger.Label()), triggerresources.Install(
				triggerName,
				brokerName,
				triggerresources.WithSubscriber(trigger.KReference(triggerName), "")))
		}
	}

	f.Assert("Broker is Ready", broker.IsReady(brokerName))
	for _, trigger := range triggers {
		triggerName := brokerName + "-" + trigger.ShortLabel()
		f.Assert(fmt.Sprintf("Trigger %s is Ready", trigger.Label()), triggerresources.IsReady(triggerName))
	}
	return f
}

// BrokerFeatureSetWithBrokerDLS returns all combinations of Broker X DeadLetterSinks,
// each broker with all possible Triggers with the DeadLetterSink set on the Broker.
func BrokerFeatureSetWithBrokerDLS(short bool) feature.FeatureSet {
	dls := deadLetterSinks
	if short {
		dls = deadLetterSinksShort
	}
	var features []*feature.Feature
	for _, broker := range brokers {
		for _, deadLetterSink := range dls {
			features = append(features, BrokerReadiness(broker, deadLetterSink, triggers, nil))
		}
	}
	return feature.FeatureSet{
		Name:     "BrokerDLS",
		Features: features,
	}
}

// BrokerFeatureSetWithTriggerDLS returns all combinations of Broker X DeadLetterSinks,
// each broker with all possible Triggers with the DeadLetterSink set on the Trigger.
func BrokerFeatureSetWithTriggerDLS(short bool) feature.FeatureSet {
	dls := deadLetterSinks
	if short {
		dls = deadLetterSinksShort
	}
	var features []*feature.Feature
	for _, broker := range brokers {
		for _, deadLetterSink := range dls {
			features = append(features, BrokerReadiness(broker, nil, triggers, deadLetterSink))
		}
	}
	return feature.FeatureSet{
		Name:     "TriggerDLS",
		Features: features,
	}
}
