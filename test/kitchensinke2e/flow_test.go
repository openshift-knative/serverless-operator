package kitchensinke2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
	"knative.dev/reconciler-test/pkg/feature"
)

func TestFlowReadiness(t *testing.T) {
	featureSets := []feature.FeatureSet{
		features.SequenceNoReplyFeatureSet(false),
		features.ParallelNoReplyFeatureSet(false),
		features.SequenceGlobalReplyFeatureSet(false),
		features.ParallelGlobalReplyFeatureSet(false),
	}
	for _, fs := range featureSets {
		testFeatureSet(t, fs)
	}
}
