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
	kafkaSink,
	jobSink,
	eventTransformSink,
}

var sinksShort = []component{
	ksvc,
}

// sinksLight is used when deploying multiple instances of the sink
// to reduce CPU/Mem requirements.
var sinksLight = []component{
	inMemoryChannel,
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
	triggersShort        = sinksShort
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
				triggerresources.WithBrokerName(brokerName),
				triggerresources.WithSubscriber(trigger.KReference(triggerName), ""),
				triggerresources.WithDeadLetterSink(triggerDls.KReference(triggerDlsName), "")))
		} else {
			f.Setup(fmt.Sprintf("Install a %s Trigger", trigger.Label()), triggerresources.Install(
				triggerName,
				triggerresources.WithBrokerName(brokerName),
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

func BrokerFeatureSetWithBrokerDLS(version string) feature.FeatureSet {
	return brokerFeatureSetWithBrokerDLS(false, 1, version)
}

func BrokerFeatureSetWithBrokerDLSShort(version string) feature.FeatureSet {
	return brokerFeatureSetWithBrokerDLS(true, 1, version)
}

func BrokerFeatureSetWithBrokerDLSStress(version string) feature.FeatureSet {
	return brokerFeatureSetWithBrokerDLS(true, NumDeployments, version)
}

// BrokerFeatureSetWithBrokerDLS returns all combinations of Broker X DeadLetterSinks,
// each broker with all possible Triggers with the DeadLetterSink set on the Broker.
func brokerFeatureSetWithBrokerDLS(short bool, times int, version string) feature.FeatureSet {
	dls := deadLetterSinks
	trgs := triggers
	if short {
		dls = deadLetterSinksShort
		trgs = triggersShort
	}
	brks := filterByVersion(brokers, version)
	dls = filterByVersion(dls, version)
	trgs = filterByVersion(trgs, version)
	features := make([]*feature.Feature, 0, len(brks)*len(dls))
	for _, broker := range brks {
		for _, deadLetterSink := range dls {
			for i := 0; i < times; i++ {
				features = append(features, BrokerReadiness(i, broker, deadLetterSink, trgs, nil))
			}
		}
	}
	return feature.FeatureSet{
		Name:     "BrokerDLS",
		Features: features,
	}
}

func BrokerFeatureSetWithTriggerDLS(version string) feature.FeatureSet {
	return brokerFeatureSetWithTriggerDLS(false, 1, version)
}

func BrokerFeatureSetWithTriggerDLSShort(version string) feature.FeatureSet {
	return brokerFeatureSetWithTriggerDLS(true, 1, version)
}

func BrokerFeatureSetWithTriggerDLSStress(version string) feature.FeatureSet {
	return brokerFeatureSetWithTriggerDLS(true, NumDeployments, version)
}

// BrokerFeatureSetWithTriggerDLS returns all combinations of Broker X DeadLetterSinks,
// each broker with all possible Triggers with the DeadLetterSink set on the Trigger.
func brokerFeatureSetWithTriggerDLS(short bool, times int, version string) feature.FeatureSet {
	dls := deadLetterSinks
	trgs := triggers
	if short {
		dls = deadLetterSinksShort
		trgs = triggersShort
	}
	brks := filterByVersion(brokers, version)
	dls = filterByVersion(dls, version)
	trgs = filterByVersion(trgs, version)
	features := make([]*feature.Feature, 0, len(brks)*len(dls))
	for _, broker := range brks {
		for _, deadLetterSink := range dls {
			for i := 0; i < times; i++ {
				features = append(features, BrokerReadiness(i, broker, nil, trgs, deadLetterSink))
			}
		}
	}
	return feature.FeatureSet{
		Name:     "TriggerDLS",
		Features: features,
	}
}
