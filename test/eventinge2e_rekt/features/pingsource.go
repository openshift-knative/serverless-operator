package features

import (
	"context"

	"github.com/cloudevents/sdk-go/v2/test"
	"github.com/openshift-knative/serverless-operator/test/eventinge2e_rekt/resources/brokerconfig"
	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/eventing/test/rekt/resources/pingsource"
	"knative.dev/eventing/test/rekt/resources/trigger"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/resources/svc"
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

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver)) //eventshub.AsKnativeService

	f.Setup("install trigger", trigger.Install(trig, br,
		trigger.WithSubscriber(svc.AsKReference(sink), "")))
	f.Setup("trigger goes ready", trigger.IsReady(trig))

	f.Setup("install pingsource", func(ctx context.Context, t feature.T) {
		// TODO: Couldn't get address ???
		brokeruri, err := broker.Address(ctx, br)
		if err != nil {
			t.Error("Failed to get address of broker", err)
		}
		pingsource.Install(source, pingsource.WithSink(nil, brokeruri.String()))(ctx, t)
	})
	f.Requirement("pingsource goes ready", pingsource.IsReady(source))

	f.Stable("pingsource as event source").
		Must("delivers events",
			assert.OnStore(sink).MatchEvent(test.HasType("dev.knative.sources.ping")).AtLeast(1))

	return f
}
