package v1alpha1

import (
	"testing"

	knativeoperatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	apistest "knative.dev/pkg/apis/testing"
)

func TestKnativeKafkaHappyPath(t *testing.T) {
	ks := &KnativeKafkaStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, knativeoperatorv1alpha1.InstallSucceeded, t)

	// Install succeeds.
	ks.MarkInstallSucceeded()
	apistest.CheckConditionSucceeded(ks, knativeoperatorv1alpha1.InstallSucceeded, t)

	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}
}

func TestKnativeKafkaErrorPath(t *testing.T) {
	ks := &KnativeKafkaStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, knativeoperatorv1alpha1.InstallSucceeded, t)

	// Install fails.
	ks.MarkInstallFailed("test")
	apistest.CheckConditionFailed(ks, knativeoperatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Install now succeeds.
	ks.MarkInstallSucceeded()
	apistest.CheckConditionSucceeded(ks, knativeoperatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}
}
