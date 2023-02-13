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
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
	"github.com/openshift-knative/serverless-operator/test/upgrade"
	"github.com/openshift-knative/serverless-operator/test/upgrade/installation"
	"knative.dev/pkg/injection"
	_ "knative.dev/pkg/system/testing"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	// Make sure to initialize flags from knative.dev/pkg/test before parsing them.
	pkgTest "knative.dev/pkg/test"
)

var global environment.GlobalEnvironment

func init() {
	// environment.InitFlags registers state and level filter flags.
	environment.InitFlags(flag.CommandLine)
}

func TestMain(m *testing.M) {
	// We get a chance to parse flags to include the framework flags for the
	// framework as well as any additional flags included in the integration.
	flag.Parse()

	// EnableInjectionOrDie will enable client injection, this is used by the
	// testing framework for namespace management, and could be leveraged by
	// features to pull Kubernetes clients or the test environment out of the
	// context passed in the features.
	cfg, err := pkgTest.Flags.ClientConfig.GetRESTConfig()
	if err != nil {
		log.Fatal("Error building client config: ", err)
	}
	ctx, startInformers := injection.EnableInjectionOrDie(nil, cfg) //nolint
	startInformers()

	// global is used to make instances of Environments, NewGlobalEnvironment
	// is passing and saving the client injection enabled context for use later.
	global = environment.NewGlobalEnvironment(ctx)

	// Run the tests.
	os.Exit(m.Run())
}

func TestKitchensink(t *testing.T) {
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
			PreUpgrade:  featureGroup.PreUpgradeTests(),
			PostUpgrade: featureGroup.PostUpgradeTests(),
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith: []pkgupgrade.Operation{
				pkgupgrade.NewOperation("UpgradeServerless", func(c pkgupgrade.Context) {
					time.Sleep(10 * time.Second)
				}),
			},
		},
	}
	suite.Execute(cfg)
}
