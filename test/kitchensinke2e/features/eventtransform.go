package features

import (
	"fmt"

	eventingv1alpha1 "knative.dev/eventing/pkg/apis/eventing/v1alpha1"
	"knative.dev/eventing/test/rekt/resources/eventtransform"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/reconciler-test/pkg/feature"
)

var jsonataSample = `
{
  "specversion": "1.0",
  "id": id,
  "type": "event.extracted",
  "source": source,
  "reason": data.reason,
  "data": $
}
`

var eventTransformers = []component{
	eventTransformGeneric,
}

func EventTransformTriggerReadiness(index int, eventTr component, sink component, replyJsonata bool) *feature.Feature {

	replyLabel := ""
	if replyJsonata {
		replyLabel = "reply"
	}
	testLabel := shortLabel(eventTr) + replyLabel + shortLabel(sink)
	testLabel = fmt.Sprintf("%s-%d", testLabel, index)

	eventTrName := testLabel
	sinkName := testLabel + "-sink"

	f := feature.NewFeatureNamed(fmt.Sprintf("%s with %s as sink", label(eventTr)+replyLabel, label(sink)))

	f.Setup("Install Sink", sink.Install(sinkName))
	f.Setup("Install EventTransform", eventTr.Install(eventTrName,
		eventtransform.WithSpec(
			eventtransform.WithSink(&duckv1.Destination{Ref: sink.KReference(sinkName)}),
			eventtransform.WithJsonata(eventingv1alpha1.JsonataEventTransformationSpec{Expression: jsonataSample}),
			func(spec *eventingv1alpha1.EventTransformSpec) {
				if replyJsonata {
					if spec.Reply == nil {
						spec.Reply = &eventingv1alpha1.ReplySpec{}
					}
					spec.Reply.Jsonata = &eventingv1alpha1.JsonataEventTransformationSpec{Expression: jsonataSample}
				}
			},
		)))

	f.Assert("Sink is Ready", sink.IsReady(sinkName))
	f.Assert("EventTransform is Ready", eventTr.IsReady(eventTrName))

	return f
}

func EventTransformFeatureSet() feature.FeatureSet {
	return eventTransformFeatureSet(false, 1)
}

// eventTransformFeatureSet return combinations of eventTransform (with and without reply) for each sink
func eventTransformFeatureSet(short bool, times int) feature.FeatureSet {

	sinks := sinksAll
	if short {
		sinks = sinksShort
	}
	if times > 1 {
		sinks = sinksLight
	}
	features := make([]*feature.Feature, 0, len(eventTransformers)*len(sinks)*times*2)
	for _, eventTr := range eventTransformers {
		for _, sink := range sinks {
			for i := 0; i < times; i++ {
				features = append(features, EventTransformTriggerReadiness(i, eventTr, sink, true))
				features = append(features, EventTransformTriggerReadiness(i, eventTr, sink, false))
			}
		}
	}

	return feature.FeatureSet{
		Name:     "EventTransform",
		Features: features,
	}
}
