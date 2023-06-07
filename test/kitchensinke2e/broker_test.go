package kitchensinke2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
)

func TestBrokerReadinessBrokerDLS(t *testing.T) {
	featureSet := features.BrokerFeatureSetWithBrokerDLS(false)
	for _, f := range featureSet.Features {
		f := f
		t.Run(featureSet.Name, func(t *testing.T) {
			t.Parallel()
			ctx, env := defaultEnvironment(t)
			env.Test(ctx, t, f)
		})
	}
}

func TestBrokerReadinessTriggerDLS(t *testing.T) {
	featureSet := features.BrokerFeatureSetWithTriggerDLS(false)
	for _, f := range featureSet.Features {
		f := f
		t.Run(featureSet.Name, func(t *testing.T) {
			t.Parallel()
			ctx, env := defaultEnvironment(t)
			env.Test(ctx, t, f)
		})
	}
}
