package kitchensinke2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
)

func TestChannelReadiness(t *testing.T) {
	testFeatureSet(t, features.ChannelFeatureSet(false))
}
