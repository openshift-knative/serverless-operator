/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"testing"

	knativeoperatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	apistest "knative.dev/pkg/apis/testing"
)

func TestKnativeKafkaHappyPath(t *testing.T) {
	ks := &KnativeKafkaStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, knativeoperatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, knativeoperatorv1alpha1.InstallSucceeded, t)

	// Install succeeds.
	ks.MarkInstallSucceeded()
	// Dependencies are assumed successful too.
	apistest.CheckConditionOngoing(ks, knativeoperatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, knativeoperatorv1alpha1.InstallSucceeded, t)

	// Deployments are not available at first.
	ks.MarkDeploymentsNotReady()
	apistest.CheckConditionFailed(ks, knativeoperatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, knativeoperatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Deployments become ready and we're good.
	ks.MarkDeploymentsAvailable()
	apistest.CheckConditionSucceeded(ks, knativeoperatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, knativeoperatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}
}

func TestKnativeKafkaErrorPath(t *testing.T) {
	ks := &KnativeKafkaStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, knativeoperatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, knativeoperatorv1alpha1.InstallSucceeded, t)

	// Install fails.
	ks.MarkInstallFailed("test")
	apistest.CheckConditionOngoing(ks, knativeoperatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionFailed(ks, knativeoperatorv1alpha1.InstallSucceeded, t)

	// Install now succeeds.
	ks.MarkInstallSucceeded()
	apistest.CheckConditionOngoing(ks, knativeoperatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, knativeoperatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Deployments become ready
	ks.MarkDeploymentsAvailable()
	apistest.CheckConditionSucceeded(ks, knativeoperatorv1alpha1.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, knativeoperatorv1alpha1.InstallSucceeded, t)
	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}
}
