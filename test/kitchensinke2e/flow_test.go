package kitchensinke2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
	"knative.dev/reconciler-test/pkg/feature"
)

func TestFlowReadiness(t *testing.T) {
	featureSets := []feature.FeatureSet{
		features.SequenceNoReplyFeatureSet(),
		features.ParallelNoReplyFeatureSet(),
		features.SequenceGlobalReplyFeatureSet(),
		features.ParallelGlobalReplyFeatureSet(),
	}

	for _, fs := range featureSets {
		for _, f := range fs.Features {
			f := f
			t.Run(fs.Name, func(t *testing.T) {
				t.Parallel()
				ctx, env := defaultEnvironment(t)
				env.Test(ctx, t, f)
			})
		}
	}
}
