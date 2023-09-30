package soake2erekt

import (
	"context"
	"fmt"
	"testing"

	cetest "github.com/cloudevents/sdk-go/v2/test"
	testpkg "knative.dev/eventing-kafka-broker/test/pkg"
	"knative.dev/eventing-kafka-broker/test/rekt/resources/kafkasink"
	"knative.dev/eventing-kafka-broker/test/rekt/resources/kafkasource"
	"knative.dev/eventing-kafka-broker/test/rekt/resources/kafkatopic"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/manifest"
	"knative.dev/reconciler-test/pkg/resources/service"
)

/*
The scenarios define names of all resources used in the scenario, so they can reference other resources
in a typesafe-ish kind of way
*/
type kafkaSourceScenarioNames struct {
	receiver    string
	sender      string
	kafkaTopic  string
	kafkaSink   string
	kafkaSource string
}

func eventshubReceiverFeature(names kafkaSourceScenarioNames) *feature.Feature {
	f := feature.NewFeatureNamed("receiver-create")
	f.Setup("install eventshub receiver", eventshub.Install(names.receiver, eventshub.StartReceiver))
	return f
}

func kafkaSourceScenarioTopicAndSinkSetupFeature(names kafkaSourceScenarioNames) *feature.Feature {
	f := feature.NewFeatureNamed("kafka-topic-and-sink-setup")
	f.Setup("install kafka topic", kafkatopic.Install(names.kafkaTopic))
	f.Setup("topic is ready", kafkatopic.IsReady(names.kafkaTopic))

	f.Setup("install kafkasink", kafkasink.Install(names.kafkaSink, names.kafkaTopic,
		testpkg.BootstrapServersPlaintextArr))
	f.Setup("kafkasink is ready", kafkasink.IsReady(names.kafkaSink))

	return f
}

func kafkaSourceScenarioInstallKafkaSourceFeature(names kafkaSourceScenarioNames) *feature.Feature {
	f := feature.NewFeatureNamed("kafka-source-add")

	kafkaSourceOpts := []manifest.CfgFn{
		kafkasource.WithSink(service.AsKReference(names.receiver), ""),
		kafkasource.WithTopics([]string{names.kafkaTopic}),
		kafkasource.WithBootstrapServers(testpkg.BootstrapServersPlaintextArr),
		kafkasource.WithConsumers(7),
	}

	f.Setup("install kafka source", kafkasource.Install(names.kafkaSource, kafkaSourceOpts...))
	return f
}

func kafkaSourceScenarioIsReadyKafkaSourceFeature(names kafkaSourceScenarioNames) *feature.Feature {
	f := feature.NewFeatureNamed("kafka-source-isready")
	f.Setup("kafka source is ready", kafkasource.IsReady(names.kafkaSource))
	return f
}

func matchEvent(sink string, matcher cetest.EventMatcher, exact int) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		assert.OnStore(sink).MatchEvent(matcher).Exact(exact)(ctx, t)
	}
}

func kafkaSinkSendFeature(names kafkaSourceScenarioNames) *feature.Feature {
	f := feature.NewFeatureNamed("kafka-sink-send")

	e := cetest.FullEvent()
	e.SetData("text/json", names.sender)

	f.Requirement("install eventshub sender", eventshub.Install(names.sender,
		eventshub.StartSenderToResource(kafkasink.GVR(), names.kafkaSink),
		eventshub.InputEvent(e)),
	)

	return f
}

func verifyEventReceivedFeature(names kafkaSourceScenarioNames, eventsToBeReceived int) *feature.Feature {
	f := feature.NewFeatureNamed("verify-events-received")

	e := cetest.FullEvent()
	e.SetData("text/json", names.sender)

	matcher := cetest.HasData(e.Data())
	f.Assert("eventshub receiver gets events", matchEvent(names.receiver, matcher, eventsToBeReceived))

	return f
}

func verifySingleEventReceivedFeature(names kafkaSourceScenarioNames) *feature.Feature {
	return verifyEventReceivedFeature(names, 1)
}

/*
TestKafkaSourceStableSoak on each iteration,
sends an event to a kafkasource
verifies an event is received
*/
func TestKafkaSourceStableSoak(t *testing.T) {
	t.Parallel()

	topicPrefix := feature.MakeRandomK8sName("topic")
	kafkaSinkPrefix := topicPrefix
	kafkaSourceName := feature.MakeRandomK8sName("kafkaSource")
	senderPrefix := feature.MakeRandomK8sName("sender")

	names := func(copyID, iteration int) kafkaSourceScenarioNames {
		return kafkaSourceScenarioNames{
			receiver:    "receiver",
			sender:      fmt.Sprintf("%s-%d-%d", senderPrefix, copyID, iteration),
			kafkaTopic:  fmt.Sprintf("%s-%d", topicPrefix, copyID),
			kafkaSink:   fmt.Sprintf("%s-%d", kafkaSinkPrefix, copyID),
			kafkaSource: kafkaSourceName,
		}
	}

	namesFn := func(fn func(kafkaSourceScenarioNames) *feature.Feature) SoakFn {
		return func(copyID, iteration int) *feature.Feature {
			return fn(names(copyID, iteration))
		}
	}

	soakTest := SoakTest{
		NamespacePrefix: "test-kafka-source-stable-",
		SetupFns: []SoakFn{
			namesFn(eventshubReceiverFeature),
			namesFn(kafkaSourceScenarioTopicAndSinkSetupFeature),
			namesFn(kafkaSourceScenarioInstallKafkaSourceFeature),
			namesFn(kafkaSourceScenarioIsReadyKafkaSourceFeature),
		},
		IterationFns: []SoakFn{
			namesFn(kafkaSinkSendFeature),
			namesFn(verifySingleEventReceivedFeature),
		},
	}

	RunSoakTestWithDefaultCopies(t, soakTest)
}

/*
TestKafkaSourceRecreateSoak on each iteration,
creates a kafkasources from a single topic
sends an event
verifies an event is received
*/
func TestKafkaSourceRecreateSoak(t *testing.T) {
	t.Parallel()

	topicPrefix := feature.MakeRandomK8sName("topic")
	kafkaSinkPrefix := topicPrefix
	kafkaSourcePrefix := feature.MakeRandomK8sName("kafkaSource")
	senderPrefix := feature.MakeRandomK8sName("sender")

	namesFn := func(fn func(kafkaSourceScenarioNames) *feature.Feature) SoakFn {
		return func(copyID, iteration int) *feature.Feature {
			return fn(kafkaSourceScenarioNames{
				receiver: "receiver",
				sender:   fmt.Sprintf("%s-%d-%d", senderPrefix, copyID, iteration),

				kafkaTopic:  fmt.Sprintf("%s-%d", topicPrefix, copyID),
				kafkaSink:   fmt.Sprintf("%s-%d", kafkaSinkPrefix, copyID),
				kafkaSource: fmt.Sprintf("%s-%d", kafkaSourcePrefix, iteration),
			})
		}
	}

	soakTest := SoakTest{
		NamespacePrefix: "test-kafka-source-recreate-",
		SetupFns: []SoakFn{
			namesFn(eventshubReceiverFeature),
			namesFn(kafkaSourceScenarioTopicAndSinkSetupFeature),
		},
		IterationFns: []SoakFn{
			namesFn(kafkaSourceScenarioInstallKafkaSourceFeature),
			namesFn(kafkaSourceScenarioIsReadyKafkaSourceFeature),
			namesFn(kafkaSinkSendFeature),
			namesFn(verifySingleEventReceivedFeature),
		},
	}

	RunSoakTestWithDefaultCopies(t, soakTest)
}

/*
TestKafkaSourceAddingAndRemovingSoak on each iteration,
creates 16 kafkasources from a single topic
sends an event
verifies an event is received 16 times
*/
func TestKafkaSourceAddingAndRemovingSoak(t *testing.T) {
	t.Parallel()

	topicPrefix := feature.MakeRandomK8sName("topic")
	kafkaSinkPrefix := topicPrefix
	kafkaSourcePrefix := feature.MakeRandomK8sName("kafkaSource")
	senderPrefix := feature.MakeRandomK8sName("sender")

	const max = 16

	names := func(copyID, iteration int) kafkaSourceScenarioNames {
		return kafkaSourceScenarioNames{
			receiver:   "receiver",
			kafkaTopic: fmt.Sprintf("%s-%d", topicPrefix, copyID),
			kafkaSink:  fmt.Sprintf("%s-%d", kafkaSinkPrefix, copyID),
			// kafkaSource:  not used, generated below dynamically for each of the 16 kafkasources
			sender: fmt.Sprintf("%s-%d-%d", senderPrefix, copyID, iteration),
		}
	}

	namesFn := func(fn func(kafkaSourceScenarioNames) *feature.Feature) func(copyID, iteration int) *feature.Feature {
		return func(copyID, iteration int) *feature.Feature {
			return fn(names(copyID, iteration))
		}
	}

	// As part of this soak test, we crate 16 kafkasources, then wait for them to be ready,
	// and finally send an event and verify an event was received 16 times
	iterationFuncs := make([]SoakFn, max*2+2)
	for j := 0; j < max; j++ {
		j := j

		iterationFuncs[j] = func(copyID, iteration int) *feature.Feature {
			ns := names(copyID, iteration)
			ns.kafkaSource = fmt.Sprintf("%s-%d-%d", kafkaSourcePrefix, iteration, j)
			return kafkaSourceScenarioInstallKafkaSourceFeature(ns)
		}
		iterationFuncs[max+j] = func(copyID, iteration int) *feature.Feature {
			ns := names(copyID, iteration)
			ns.kafkaSource = fmt.Sprintf("%s-%d-%d", kafkaSourcePrefix, iteration, j)
			return kafkaSourceScenarioIsReadyKafkaSourceFeature(ns)
		}
	}
	iterationFuncs[2*max] = namesFn(kafkaSinkSendFeature)
	iterationFuncs[2*max+1] = func(copyID, iteration int) *feature.Feature {
		return verifyEventReceivedFeature(names(copyID, iteration), max)
	}

	soakTest := SoakTest{
		NamespacePrefix: "test-kafka-source-addrm-",
		SetupFns: []SoakFn{
			namesFn(eventshubReceiverFeature),
			namesFn(kafkaSourceScenarioTopicAndSinkSetupFeature),
		},
		IterationFns: iterationFuncs,
	}

	RunSoakTestWithDefaultCopies(t, soakTest)
}
