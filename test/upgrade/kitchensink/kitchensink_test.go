//go:build upgrade
// +build upgrade

/*
Copyright 2020 The Knative Authors

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

package kitchensink

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e"
	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
	"github.com/openshift-knative/serverless-operator/test/upgrade"
	"github.com/openshift-knative/serverless-operator/test/upgrade/installation"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

var global environment.GlobalEnvironment

func init() {
	// environment.InitFlags registers state and level filter flags.
	environment.InitFlags(flag.CommandLine)
}

func TestMain(m *testing.M) {
	global = environment.NewStandardGlobalEnvironment()
	// Run the tests.
	os.Exit(m.Run())
}

func TestKitchensinkUpgrade(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
	cfg := upgrade.NewUpgradeConfig(t)

	// Add here any features sets to be tested during upgrades.
	featureSets := []feature.FeatureSet{
		features.BrokerFeatureSetWithBrokerDLS(),
		features.BrokerFeatureSetWithTriggerDLS(),
	}

	var featureGroup FeatureWithEnvironmentGroup
	for _, fs := range featureSets {
		for _, f := range fs.Features {
			featureGroup = append(featureGroup, NewFeatureWithEnvironment(t, global, f))
		}
	}

	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade:    featureGroup.PreUpgradeTests(),
			PostUpgrade:   featureGroup.PostUpgradeTests(),
			PostDowngrade: featureGroup.PostDowngradeTests(),
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith:   upgrade.ServerlessUpgradeOperations(ctx),
			DowngradeWith: upgrade.ServerlessDowngradeOperations(ctx),
		},
	}
	suite.Execute(cfg)
}
