package common_test

import (
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"testing"
)

func verifyImageOverride(t *testing.T, registry *servingv1alpha1.Registry, imageName string, expected string) {
	if registry.Override[imageName] != expected {
		t.Errorf("Missing queue image. Expected a map with following override in it : %v=%v, actual: %v", imageName, expected, registry.Override)
	}
}

func verifyTimestamp(t *testing.T, annotations map[string]string) {
	if _, ok := annotations[common.MutationTimestampKey]; !ok {
		t.Errorf("Missing mutation timestamp annotation. Existing annotations: %v", annotations)
	}
}
