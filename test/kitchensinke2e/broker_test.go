package kitchensinke2e

import (
	"fmt"
	"testing"

	"knative.dev/eventing/test/rekt/resources/delivery"
	triggerresources "knative.dev/eventing/test/rekt/resources/trigger"
	"knative.dev/reconciler-test/pkg/feature"
)

var brokers = []component{
	inMemoryChannelMtBroker,
	kafkaChannelMtBroker,
	kafkaBroker,
}

var deadLetterSinks = []component{
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

var triggers = []component{
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

func BrokerFeatureSetWithBrokerDLS() feature.FeatureSet {
	var features []*feature.Feature
	// Test all combinations of Broker X DeadLetterSinks, each broker with all possible Triggers
	// with the DeadLetterSink set on the Broker
	for _, broker := range brokers {
		for _, deadLetterSink := range deadLetterSinks {
			broker := broker
			deadLetterSink := deadLetterSink
			features = append(features, BrokerReadiness(broker, deadLetterSink, triggers, nil))
		}
	}
	return feature.FeatureSet{
		Name:     "BrokerDLS",
		Features: features,
	}
}

func BrokerFeatureSetWithTriggerDLS() feature.FeatureSet {
	var features []*feature.Feature
	// Test all combinations of Broker X DeadLetterSinks, each broker with all possible Triggers
	// with the DeadLetterSink set on the Trigger
	for _, broker := range brokers {
		for _, deadLetterSink := range deadLetterSinks {
			broker := broker
			deadLetterSink := deadLetterSink
			features = append(features, BrokerReadiness(broker, nil, triggers, deadLetterSink))
		}
	}
	return feature.FeatureSet{
		Name:     "TriggerDLS",
		Features: features,
	}
}

func TestBrokerReadiness(t *testing.T) {
	featureSets := []feature.FeatureSet{
		BrokerFeatureSetWithBrokerDLS(),
		BrokerFeatureSetWithTriggerDLS(),
	}
	for _, fs := range featureSets {
		for _, f := range fs.Features {
			t.Run(fs.Name, func(t *testing.T) {
				t.Parallel()
				ctx, env := defaultContext(t)
				env.Test(ctx, t, f)
			})
		}
	}
}
