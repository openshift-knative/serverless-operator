package kitchensinke2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
	"knative.dev/reconciler-test/pkg/feature"
)

func TestBrokerReadiness(t *testing.T) {
	featureSets := []feature.FeatureSet{
		features.BrokerFeatureSetWithBrokerDLS(false),
		features.BrokerFeatureSetWithTriggerDLS(false),
	}
	for _, fs := range featureSets {
		for _, f := range fs.Features {
			f := f
			t.Run(fs.Name, func(t *testing.T) {
				t.Parallel()
				ctx, env := defaultContext(t)
				env.Test(ctx, t, f)
			})
		}
	}
}
