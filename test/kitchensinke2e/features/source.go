package features

import (
	"fmt"

	"knative.dev/eventing/test/rekt/resources/apiserversource"
	"knative.dev/reconciler-test/pkg/feature"
)

var sources = []component{
	pingSource,
	containerSource,
	apiServerSource,
	kafkaSource,
}

func SourceReadiness(index int, source component, sink component) *feature.Feature {
	testLabel := shortLabel(source) + shortLabel(sink)
	testLabel = fmt.Sprintf("%s-%d", testLabel, index)

	sourceName := testLabel
	sinkName := testLabel + "-sink"

	f := feature.NewFeatureNamed(fmt.Sprintf("%s with %s as sink", label(source), label(sink)))

	f.Setup("Install Sink", sink.Install(sinkName))
	f.Setup("Install Source", source.Install(sourceName,
		// Use apiserversource's WithSink because its template requires .sink.ref.namespace to be set.
		// This function is generic enough to work with other sources too.
		apiserversource.WithSink(sink.KReference(sinkName), "")))

	f.Assert("Sink is Ready", sink.IsReady(sinkName))
	f.Assert("Source is Ready", source.IsReady(sourceName))

	return f
}

func SourceFeatureSet() feature.FeatureSet {
	return sourceFeatureSet(false, 1)
}

func SourceFeatureSetShort() feature.FeatureSet {
	return sourceFeatureSet(true, 1)
}

func SourceFeatureSetStress() feature.FeatureSet {
	return sourceFeatureSet(true, 50)
}

// sourceFeatureSet returns all combinations of Source x Sinks.
func sourceFeatureSet(short bool, times int) feature.FeatureSet {
	sinks := sinksAll
	if short {
		sinks = sinksShort
	}
	if times > 1 {
		sinks = sinksLight
	}
	features := make([]*feature.Feature, 0, len(sources)*len(sinks))
	for _, source := range sources {
		for _, sink := range sinks {
			for i := 0; i < times; i++ {
				features = append(features, SourceReadiness(i, source, sink))
			}
		}
	}
	return feature.FeatureSet{
		Name:     "Source",
		Features: features,
	}
}
