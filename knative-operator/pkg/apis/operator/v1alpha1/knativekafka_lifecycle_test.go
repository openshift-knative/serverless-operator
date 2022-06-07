package v1alpha1

import (
	"testing"

	"knative.dev/operator/pkg/apis/operator/base"
	apistest "knative.dev/pkg/apis/testing"
)

func TestKnativeKafkaHappyPath(t *testing.T) {
	ks := &KnativeKafkaStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, base.InstallSucceeded, t)

	// Install succeeds.
	ks.MarkInstallSucceeded()
	// Dependencies are assumed successful too.
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)

	// Deployments are not available at first.
	ks.MarkDeploymentsNotReady()
	apistest.CheckConditionFailed(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Deployments become ready and we're good.
	ks.MarkDeploymentsAvailable()
	apistest.CheckConditionSucceeded(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}
}

func TestKnativeKafkaErrorPath(t *testing.T) {
	ks := &KnativeKafkaStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, base.InstallSucceeded, t)

	// Install fails.
	ks.MarkInstallFailed("test")
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionFailed(ks, base.InstallSucceeded, t)

	// Install now succeeds.
	ks.MarkInstallSucceeded()
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Deployments become ready
	ks.MarkDeploymentsAvailable()
	apistest.CheckConditionSucceeded(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}
}
