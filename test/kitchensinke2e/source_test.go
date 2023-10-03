package kitchensinke2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
)

func TestSourceReadiness(t *testing.T) {
	testFeatureSet(t, features.SourceFeatureSet())
}
