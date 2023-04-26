package features

import (
	"github.com/cloudevents/sdk-go/v2/test"
	"github.com/openshift-knative/serverless-operator/test/eventinge2e_rekt/resources/brokerconfig"
	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/eventing/test/rekt/resources/pingsource"
	"knative.dev/eventing/test/rekt/resources/trigger"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/resources/knativeservice"
)

func SendsEventsWithSinkRef() *feature.Feature {
	source := feature.MakeRandomK8sName("pingsource")
	sink := feature.MakeRandomK8sName("sink")
	config := feature.MakeRandomK8sName("config")
	trig := feature.MakeRandomK8sName("trig")
	br := feature.MakeRandomK8sName("broker")
	f := feature.NewFeature()

	f.Setup("install broker configmap", brokerconfig.Install(config,
		brokerconfig.WithGenericChannelMTBroker()))

	///broker.WithEnvConfig()...
	f.Setup("install broker", broker.Install(br, broker.WithConfig(config)))
	f.Requirement("broker is ready", broker.IsReady(br))
	f.Requirement("broker is addressable", broker.IsAddressable(br))

	f.Setup("install sink", eventshub.Install(sink,
		eventshub.StartReceiver,
		eventshub.AsKnativeService))

	f.Setup("install trigger", trigger.Install(trig, br,
		trigger.WithSubscriber(&duckv1.KReference{
			APIVersion: knativeservice.GVR().GroupVersion().String(),
			Kind:       "Service", //broker.GVR().Resource
			Name:       sink,
		}, "")))
	f.Setup("trigger goes ready", trigger.IsReady(trig))

	f.Setup("install pingsource", pingsource.Install(source, pingsource.WithSink(
		&duckv1.KReference{
			APIVersion: broker.GVR().GroupVersion().String(),
			Kind:       "Broker", //broker.GVR().Resource
			Name:       br,
		}, "")))
	f.Requirement("pingsource goes ready", pingsource.IsReady(source))

	f.Stable("pingsource as event source").
		Must("delivers events",
			assert.OnStore(sink).MatchEvent(test.HasType("dev.knative.sources.ping")).AtLeast(1))

	return f
}
