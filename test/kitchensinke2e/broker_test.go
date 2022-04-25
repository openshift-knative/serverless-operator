package kitchensinke2e

import (
	"fmt"
	"testing"

	"knative.dev/eventing/test/rekt/resources/delivery"
	triggerresources "knative.dev/eventing/test/rekt/resources/trigger"
	"knative.dev/reconciler-test/pkg/feature"
)

func BrokerReadiness(testLabel string, broker component, brokerDls component, triggers []component, triggerDls component) *feature.Feature {
	brokerName := testLabel
	dlsName := testLabel + "-dls"
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

func TestBrokerReadiness(t *testing.T) {
	brokers := []component{
		inMemoryChannelMtBroker,
		kafkaChannelMtBroker,
		kafkaBroker,
	}

	deadLetterSinks := []component{
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

	triggers := []component{
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

	// Test all combinations of Broker X DeadLetterSinks, each broker with all possible Triggers
	// with the DeadLetterSink set on the Broker
	for _, broker := range brokers {
		for _, deadLetterSink := range deadLetterSinks {
			broker := broker
			deadLetterSink := deadLetterSink

			testLabel := shortLabel(broker) + shortLabel(deadLetterSink)

			t.Run(testLabel, func(t *testing.T) {
				t.Parallel()

				ctx, env := defaultContext(t)

				env.Test(ctx, t, BrokerReadiness(testLabel, broker, deadLetterSink, triggers, nil))
			})
		}
	}

	// Test all combinations of Broker X DeadLetterSinks, each broker with all possible Triggers
	// with the DeadLetterSink set on the Trigger
	for _, broker := range brokers {
		for _, deadLetterSink := range deadLetterSinks {
			broker := broker
			deadLetterSink := deadLetterSink

			// Just to distinguish the label from the ones above, we add "t" for a "T"rigger deadLetterSink
			testLabel := shortLabel(broker) + "t" + shortLabel(deadLetterSink)

			t.Run(testLabel, func(t *testing.T) {
				t.Parallel()

				ctx, env := defaultContext(t)

				env.Test(ctx, t, BrokerReadiness(testLabel, broker, nil, triggers, deadLetterSink))
			})
		}
	}
}
