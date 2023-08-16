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

func BrokerReadiness(index int, broker component, brokerDls component, triggers []component, triggerDls component) *feature.Feature {
	testLabel := shortLabel(broker)
	if brokerDls != nil {
		testLabel = testLabel + "b" + shortLabel(brokerDls)
	}
	if triggerDls != nil {
		testLabel = testLabel + "t" + shortLabel(triggerDls)
	}

	testLabel = fmt.Sprintf("%s-%d", testLabel, index)

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

func BrokerFeatureSetWithBrokerDLS() feature.FeatureSet {
	return brokerFeatureSetWithBrokerDLS(false, 1)
}

func BrokerFeatureSetWithBrokerDLSShort() feature.FeatureSet {
	return brokerFeatureSetWithBrokerDLS(true, 1)
}

func BrokerFeatureSetWithBrokerDLSStress() feature.FeatureSet {
	return brokerFeatureSetWithBrokerDLS(true, NumDeployments)
}

// BrokerFeatureSetWithBrokerDLS returns all combinations of Broker X DeadLetterSinks,
// each broker with all possible Triggers with the DeadLetterSink set on the Broker.
func brokerFeatureSetWithBrokerDLS(short bool, times int) feature.FeatureSet {
	dls := deadLetterSinks
	if short {
		dls = deadLetterSinksShort
	}
	features := make([]*feature.Feature, 0, len(brokers)*len(dls))
	for _, broker := range brokers {
		for _, deadLetterSink := range dls {
			for i := 0; i < times; i++ {
				features = append(features, BrokerReadiness(i, broker, deadLetterSink, triggers, nil))
			}
		}
	}
	return feature.FeatureSet{
		Name:     "BrokerDLS",
		Features: features,
	}
}

func BrokerFeatureSetWithTriggerDLS() feature.FeatureSet {
	return brokerFeatureSetWithTriggerDLS(false, 1)
}

func BrokerFeatureSetWithTriggerDLSShort() feature.FeatureSet {
	return brokerFeatureSetWithTriggerDLS(true, 1)
}

func BrokerFeatureSetWithTriggerDLSStress() feature.FeatureSet {
	return brokerFeatureSetWithTriggerDLS(true, NumDeployments)
}

// BrokerFeatureSetWithTriggerDLS returns all combinations of Broker X DeadLetterSinks,
// each broker with all possible Triggers with the DeadLetterSink set on the Trigger.
func brokerFeatureSetWithTriggerDLS(short bool, times int) feature.FeatureSet {
	dls := deadLetterSinks
	if short {
		dls = deadLetterSinksShort
	}
	features := make([]*feature.Feature, 0, len(brokers)*len(dls))
	for _, broker := range brokers {
		for _, deadLetterSink := range dls {
			for i := 0; i < times; i++ {
				features = append(features, BrokerReadiness(i, broker, nil, triggers, deadLetterSink))
			}
		}
	}
	return feature.FeatureSet{
		Name:     "TriggerDLS",
		Features: features,
	}
}
