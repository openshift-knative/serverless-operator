package kitchensinke2e

import (
	"fmt"
	"testing"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
	"knative.dev/reconciler-test/pkg/feature"
)

const groupSize = 8

func TestBrokerReadinessBrokerDLS(t *testing.T) {
	testFeatureSet(t, features.BrokerFeatureSetWithBrokerDLS())
}

func TestBrokerReadinessTriggerDLS(t *testing.T) {
	testFeatureSet(t, features.BrokerFeatureSetWithTriggerDLS())
}

func split(featureSet feature.FeatureSet, groupSize int) []feature.FeatureSet {
	fss := make([]feature.FeatureSet, 0, len(featureSet.Features)/groupSize)

	var j int
	for i := 0; i < len(featureSet.Features); i += groupSize {
		j += groupSize
		if j > len(featureSet.Features) {
			// Put the remainder in the last group and don't exceed boundaries.
			j = len(featureSet.Features)
		}
		fss = append(fss, feature.FeatureSet{
			Name:     fmt.Sprintf("%s-%d", featureSet.Name, i),
			Features: featureSet.Features[i:j],
		})
	}

	return fss
}

// testFeatureSet splits large feature sets into smaller groups and tests
// them sequentially while running features within those feature sets
// in parallel (each feature in its own namespace).
func testFeatureSet(t *testing.T, featureSet feature.FeatureSet) {
	for _, fs := range split(featureSet, groupSize) {
		// Run individual feature sets sequentially.
		t.Run(fs.Name, func(t *testing.T) {
			for _, f := range fs.Features {
				f := f
				// Run features within feature sets in parallel.
				t.Run(fs.Name, func(t *testing.T) {
					t.Parallel()
					ctx, env := defaultEnvironment(t)
					env.Test(ctx, t, f)
				})
			}
		})
	}
}
