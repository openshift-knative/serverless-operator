package soake2erekt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/reconciler-test/pkg/environment"

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

func matchEvent(sink string, matcher cetest.EventMatcher, atLeast int) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		assert.OnStore(sink).MatchEvent(matcher).AtLeast(atLeast)(ctx, t)
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
verifyNoKafkaSourceLeftInDispatcherConfigMap verifies there are no mentions of kafkasources in the dispatcher configmaps
that belonged to the test namespace. This test relies on the implementation detail of the dispatcher configmaps structure.
*/
func verifyNoKafkaSourceLeftInDispatcherConfigMap() *feature.Feature {
	f := feature.NewFeatureNamed("verify-dispatcher-cm-clean")
	f.Assert("no source from test namespace left in dispatcher cm", func(ctx context.Context, t feature.T) {
		ns := environment.FromContext(ctx).Namespace()
		cmlist, err := kubeclient.Get(ctx).CoreV1().ConfigMaps("knative-eventing").List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Errorf("error listing knative-eventing configmaps: %v", err)
			return
		}

		for _, cm := range cmlist.Items {
			if strings.HasPrefix(cm.Name, "kafka-source-dispatcher-") {

				dataBytes := cm.BinaryData["data"]
				if dataBytes != nil {
					var u map[string]interface{}
					err = json.Unmarshal(dataBytes, &u)
					if err != nil {
						t.Errorf("error unmarshalling %s configmap: %v", cm.Name, err)
						return
					}

					uResources, found, err := unstructured.NestedSlice(u, "resources")
					if !found {
						// could be still empty?
						continue
					}
					if err != nil {
						t.Errorf("error getting .resources from %s configmap: %v", cm.Name, err)
					}

					for _, uResource := range uResources {
						uResourceMap, ok := uResource.(map[string]interface{})
						if !ok {
							t.Errorf("unexpected type of a `resources` item in %s configmap", cm.Name)
							continue
						}
						uNamespace, found, err := unstructured.NestedString(uResourceMap, "reference", "namespace")
						if !found {
							// could be not set?
							continue
						}
						if err != nil {
							t.Errorf("error getting .reference.namespace from %s configmap: %v", cm.Name, err)
						}

						// t.Logf("XXX reference namespace: %s", uNamespace)

						if uNamespace == ns {
							t.Errorf("Found reference to a resource in the test namespace %q in the %s configmap", uNamespace, cm.Name)
						}
					}
				}
			}
		}
	})

	return f
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

	namesFn := func(env SoakEnv) kafkaSourceScenarioNames {
		return kafkaSourceScenarioNames{
			receiver:    "receiver",
			sender:      fmt.Sprintf("%s-%d-%d", senderPrefix, env.CopyID(), env.Iteration()),
			kafkaTopic:  fmt.Sprintf("%s-%d", topicPrefix, env.CopyID()),
			kafkaSink:   fmt.Sprintf("%s-%d", kafkaSinkPrefix, env.CopyID()),
			kafkaSource: kafkaSourceName,
		}
	}

	soakTest := SoakTest{
		NamespacePrefix: "test-kafka-source-stable-",
		SetupFn: func(ctx context.Context, env environment.Environment, t *testing.T) {
			// TODO: These are actually equivalent to just env.Test(ctx, t, eventshubReceiverFeature(namesFn(SoakEnvFromContext(ctx)))) , so not sure if we should bother...
			RunSoakFeatureFnWithMapping(ctx, env, t, eventshubReceiverFeature, namesFn)
			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSourceScenarioTopicAndSinkSetupFeature, namesFn)
			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSourceScenarioInstallKafkaSourceFeature, namesFn)
			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSourceScenarioIsReadyKafkaSourceFeature, namesFn)
		},
		IterationFn: func(ctx context.Context, env environment.Environment, t *testing.T) {
			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSinkSendFeature, namesFn)
			RunSoakFeatureFnWithMapping(ctx, env, t, verifySingleEventReceivedFeature, namesFn)

			// we just want to verify the source can send/receive events throughout the soak test, so let it rest here for a while
			time.Sleep(1 * time.Second)
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

	namesFn := func(env SoakEnv) kafkaSourceScenarioNames {
		return kafkaSourceScenarioNames{
			receiver:    "receiver",
			sender:      fmt.Sprintf("%s-%d-%d", senderPrefix, env.CopyID(), env.Iteration()),
			kafkaTopic:  fmt.Sprintf("%s-%d", topicPrefix, env.CopyID()),
			kafkaSink:   fmt.Sprintf("%s-%d", kafkaSinkPrefix, env.CopyID()),
			kafkaSource: fmt.Sprintf("%s-%d", kafkaSourcePrefix, env.Iteration()),
		}
	}

	soakTest := SoakTest{
		NamespacePrefix: "test-kafka-source-recreate-",
		SetupFn: func(ctx context.Context, env environment.Environment, t *testing.T) {
			RunSoakFeatureFnWithMapping(ctx, env, t, eventshubReceiverFeature, namesFn)
			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSourceScenarioTopicAndSinkSetupFeature, namesFn)
		},
		IterationFn: func(ctx context.Context, env environment.Environment, t *testing.T) {
			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSourceScenarioInstallKafkaSourceFeature, namesFn)
			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSourceScenarioIsReadyKafkaSourceFeature, namesFn)
			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSinkSendFeature, namesFn)
			RunSoakFeatureFnWithMapping(ctx, env, t, verifySingleEventReceivedFeature, namesFn)
		},
		TeardownFn: func(ctx context.Context, env environment.Environment, t *testing.T) {
			f := verifyNoKafkaSourceLeftInDispatcherConfigMap()
			env.Test(ctx, t, f)
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

	// Number of kafkasources to create and delete at once on each iteration
	const max = 16

	namesFn := func(env SoakEnv) kafkaSourceScenarioNames {
		return kafkaSourceScenarioNames{
			receiver:   "receiver",
			sender:     fmt.Sprintf("%s-%d-%d", senderPrefix, env.CopyID(), env.Iteration()),
			kafkaTopic: fmt.Sprintf("%s-%d", topicPrefix, env.CopyID()),
			kafkaSink:  fmt.Sprintf("%s-%d", kafkaSinkPrefix, env.CopyID()),
			// kafkaSource:  not used, generated below dynamically for each of the 16 kafkasources
		}
	}

	soakTest := SoakTest{
		NamespacePrefix: "test-kafka-source-addrm-",
		SetupFn: func(ctx context.Context, env environment.Environment, t *testing.T) {
			RunSoakFeatureFnWithMapping(ctx, env, t, eventshubReceiverFeature, namesFn)
			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSourceScenarioTopicAndSinkSetupFeature, namesFn)
		},
		// As part of this soak test, we crate 16 kafkasources, then wait for them to be ready,
		// and finally send an event and verify an event was received 16 times
		IterationFn: func(ctx context.Context, env environment.Environment, t *testing.T) {
			for j := 0; j < max; j++ {
				soakEnv := SoakEnvFromContext(ctx)
				names := namesFn(soakEnv)
				names.kafkaSource = fmt.Sprintf("%s-%d-%d", kafkaSourcePrefix, soakEnv.Iteration(), j)

				f := kafkaSourceScenarioInstallKafkaSourceFeature(names)
				env.Test(ctx, t, f)
			}

			for j := 0; j < max; j++ {
				soakEnv := SoakEnvFromContext(ctx)
				names := namesFn(soakEnv)
				names.kafkaSource = fmt.Sprintf("%s-%d-%d", kafkaSourcePrefix, soakEnv.Iteration(), j)

				f := kafkaSourceScenarioIsReadyKafkaSourceFeature(names)
				env.Test(ctx, t, f)
			}

			RunSoakFeatureFnWithMapping(ctx, env, t, kafkaSinkSendFeature, namesFn)

			f := verifyEventReceivedFeature(namesFn(SoakEnvFromContext(ctx)), max)
			env.Test(ctx, t, f)
		},
		TeardownFn: func(ctx context.Context, env environment.Environment, t *testing.T) {
			f := verifyNoKafkaSourceLeftInDispatcherConfigMap()
			env.Test(ctx, t, f)
		},
	}

	RunSoakTestWithDefaultCopies(t, soakTest)
}