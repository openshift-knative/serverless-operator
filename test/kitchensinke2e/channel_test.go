package kitchensinke2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
)

func TestChannelReadiness(t *testing.T) {
	featureSet := features.ChannelFeatureSet(false)
	for _, f := range featureSet.Features {
		f := f
		t.Run(featureSet.Name, func(t *testing.T) {
			t.Parallel()
			ctx, env := defaultContext(t)
			env.Test(ctx, t, f)
		})
	}
}
