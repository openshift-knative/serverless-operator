package v1alpha1

import (
	"testing"

	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	apistest "knative.dev/pkg/apis/testing"
)

func TestKnativeKafkaHappyPath(t *testing.T) {
	ks := &KnativeKafkaStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, operatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, operatorv1alpha1.InstallSucceeded, t)

	// Install succeeds.
	ks.MarkInstallSucceeded()
	// Dependencies are assumed successful too.
	apistest.CheckConditionOngoing(ks, operatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, StatefulSetsAvailable, t)
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Deployments are not available at first.
	ks.MarkDeploymentsNotReady()
	apistest.CheckConditionFailed(ks, operatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, StatefulSetsAvailable, t)
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Deployments become ready and we're good.
	ks.MarkDeploymentsAvailable()
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, StatefulSetsAvailable, t)
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	ks.MarkStatefulSetsAvailable()
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.InstallSucceeded, t)
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.InstallSucceeded, t)

	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}

}

func TestKnativeKafkaErrorPath(t *testing.T) {
	ks := &KnativeKafkaStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, operatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, operatorv1alpha1.InstallSucceeded, t)

	// Install fails.
	ks.MarkInstallFailed("test")
	apistest.CheckConditionOngoing(ks, operatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionFailed(ks, operatorv1alpha1.InstallSucceeded, t)

	// Install now succeeds.
	ks.MarkInstallSucceeded()
	apistest.CheckConditionOngoing(ks, operatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	ks.MarkStatefulSetNotReady()
	apistest.CheckConditionFailed(ks, StatefulSetsAvailable, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Deployments become ready
	ks.MarkDeploymentsAvailable()
	ks.MarkStatefulSetsAvailable()
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, StatefulSetsAvailable, t)
	apistest.CheckConditionSucceeded(ks, operatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}
}
