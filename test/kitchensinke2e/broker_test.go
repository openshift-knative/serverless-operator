package kitchensinke2e

import (
	"fmt"
	"testing"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
	"knative.dev/reconciler-test/pkg/feature"
)

const numParts = 5

func TestBrokerReadinessBrokerDLS(t *testing.T) {
	testFeatureSet(t, features.BrokerFeatureSetWithBrokerDLS(false))
}

func TestBrokerReadinessTriggerDLS(t *testing.T) {
	testFeatureSet(t, features.BrokerFeatureSetWithTriggerDLS(false))
}

func split(featureSet feature.FeatureSet, parts int) []feature.FeatureSet {
	fss := make([]feature.FeatureSet, parts)

	size := len(featureSet.Features) / parts
	var j int
	for i := 0; i < len(featureSet.Features); i += size {
		j += size
		if j+size > len(featureSet.Features) {
			// Squeeze the remainder into the last group.
			fss = append(fss, feature.FeatureSet{
				Name:     fmt.Sprintf("%s-%d", featureSet.Name, i),
				Features: featureSet.Features[i:],
			})
			break
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
	for _, fs := range split(featureSet, numParts) {
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
