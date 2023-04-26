package features

import (
	"github.com/cloudevents/sdk-go/v2/test"
	"github.com/openshift-knative/serverless-operator/test/eventinge2e_rekt/resources/brokerconfig"
	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/eventing/test/rekt/resources/pingsource"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/resources/svc"
)

const (
	cmName = "smoke-test-br-cm"
)

func SendsEventsWithSinkRef() *feature.Feature {
	source := feature.MakeRandomK8sName("pingsource")
	sink := feature.MakeRandomK8sName("sink")
	f := feature.NewFeature()

	brokerName := feature.MakeRandomK8sName("broker")

	f.Setup("install broker configmap", brokerconfig.Install("brokerconfig",
		brokerconfig.WithGenericChannelMTBroker()))

	///broker.WithEnvConfig()...
	f.Setup("install broker", broker.Install(brokerName, broker.WithConfig(cmName)))
	f.Requirement("broker is ready", broker.IsReady(brokerName))
	f.Requirement("broker is addressable", broker.IsAddressable(brokerName))

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver))

	f.Setup("install pingsource", pingsource.Install(source, pingsource.WithSink(svc.AsKReference(sink), "")))
	f.Requirement("pingsource goes ready", pingsource.IsReady(source))

	f.Stable("pingsource as event source").
		Must("delivers events",
			assert.OnStore(sink).MatchEvent(test.HasType("dev.knative.sources.ping")).AtLeast(1))

	return f
}
