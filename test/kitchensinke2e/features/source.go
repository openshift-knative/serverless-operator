package features

import (
	"fmt"

	"knative.dev/eventing/test/rekt/resources/apiserversource"
	duckv1 "knative.dev/pkg/apis/duck/v1"
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
		apiserversource.WithSink(&duckv1.Destination{Ref: sink.KReference(sinkName)})))

	f.Assert("Sink is Ready", sink.IsReady(sinkName))
	f.Assert("Source is Ready", source.IsReady(sourceName))

	return f
}

func SourceFeatureSet(version string) feature.FeatureSet {
	return sourceFeatureSet(false, 1, version)
}

func SourceFeatureSetShort(version string) feature.FeatureSet {
	return sourceFeatureSet(true, 1, version)
}

func SourceFeatureSetStress(version string) feature.FeatureSet {
	return sourceFeatureSet(true, 50, version)
}

// sourceFeatureSet returns all combinations of Source x Sinks.
func sourceFeatureSet(short bool, times int, version string) feature.FeatureSet {
	sinks := sinksAll
	if short {
		sinks = sinksShort
	}
	if times > 1 {
		sinks = sinksLight
	}
	srcs := filterByVersion(sources, version)
	sinks = filterByVersion(sinks, version)
	features := make([]*feature.Feature, 0, len(srcs)*len(sinks))
	for _, source := range srcs {
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
