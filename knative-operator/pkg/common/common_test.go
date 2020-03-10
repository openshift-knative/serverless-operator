package common_test

import (
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"testing"
)

func verifyImageOverride(t *testing.T, registry *servingv1alpha1.Registry, imageName string, expected string) {
	if registry.Override[imageName] != expected {
		t.Errorf("Missing queue image. Expected a map with following override in it : %v=%v, actual: %v", imageName, expected, registry.Override)
	}
}
