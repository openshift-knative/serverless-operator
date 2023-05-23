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

package upgrade_test

import (
	"testing"

	kafkabrokerupgrade "knative.dev/eventing-kafka-broker/test/upgrade"
	eventingupgrade "knative.dev/eventing/test/upgrade"
	_ "knative.dev/pkg/system/testing"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	servingupgrade "knative.dev/serving/test/upgrade"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/upgrade"
	"github.com/openshift-knative/serverless-operator/test/upgrade/installation"
)

// FIXME: https://github.com/knative/eventing/issues/5176 `*-config.toml` in
//        this directory are required, so that kafkaupgrade tests will see them.

func TestServerlessUpgradePrePost(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade:    preUpgradeTests(),
			PostUpgrade:   postUpgradeTests(ctx, true),
			PostDowngrade: postDowngradeTests(),
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith:   upgrade.ServerlessUpgradeOperations(ctx),
			DowngradeWith: upgrade.ServerlessDowngradeOperations(ctx),
		},
	}
	suite.Execute(pkgupgrade.Configuration{T: t})
}

func TestServerlessUpgradeContinual(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			Continual: merge(
				[]pkgupgrade.BackgroundOperation{
					servingupgrade.ProbeTest(),
					servingupgrade.AutoscaleSustainingWithTBCTest(),
					servingupgrade.AutoscaleSustainingTest(),
				},
				kafkabrokerupgrade.ChannelContinualTests(),
				kafkabrokerupgrade.BrokerContinualTests(),
				kafkabrokerupgrade.SinkContinualTests(),
			),
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith:   upgrade.ServerlessUpgradeOperations(ctx),
			DowngradeWith: upgrade.ServerlessDowngradeOperations(ctx),
		},
	}
	suite.Execute(pkgupgrade.Configuration{T: t})
}

func TestClusterUpgrade(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	if !test.Flags.UpgradeOpenShift {
		t.Skip("Cluster upgrade tests disabled unless enabled by a flag.")
	}
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade:  preUpgradeTests(),
			PostUpgrade: postUpgradeTests(ctx, false),
			// Do not include continual tests as they're failing across cluster upgrades.
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith: []pkgupgrade.Operation{
				pkgupgrade.NewOperation("OpenShift Upgrade", func(c pkgupgrade.Context) {
					upgradeFunc := installation.UpgradeOpenShift
					eus, err := installation.IsChannelEUS(ctx)
					if err != nil {
						c.T.Error("Unable to detect EUS upgrade:", err)
					}
					if eus {
						upgradeFunc = installation.UpgradeEUS
					}
					if err := upgradeFunc(ctx); err != nil {
						c.T.Error("OpenShift upgrade failed:", err)
					}
				}),
			},
		},
	}
	suite.Execute(pkgupgrade.Configuration{T: t})
}

func merge(slices ...[]pkgupgrade.BackgroundOperation) []pkgupgrade.BackgroundOperation {
	l := 0
	for _, slice := range slices {
		l += len(slice)
	}
	result := make([]pkgupgrade.BackgroundOperation, 0, l)
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

func preUpgradeTests() []pkgupgrade.Operation {
	tests := []pkgupgrade.Operation{
		eventingupgrade.PreUpgradeTest(),
		kafkabrokerupgrade.ChannelPreUpgradeTest(),
		kafkabrokerupgrade.SourcePreUpgradeTest(glob),
		kafkabrokerupgrade.BrokerPreUpgradeTest(),
		kafkabrokerupgrade.SinkPreUpgradeTest(),
	}
	// We might want to skip pre-upgrade test if we want to re-use the services
	// from the previous run. For example, to let them survive both Serverless
	// and OCP upgrades. This allows for more variants of tests, with different
	// order of upgrades.
	if test.Flags.SkipServingPreUpgrade {
		return tests
	}
	return append(tests, servingupgrade.ServingPreUpgradeTests()...)
}

func postUpgradeTests(ctx *test.Context, failOnNoJobs bool) []pkgupgrade.Operation {
	tests := []pkgupgrade.Operation{waitForServicesReady(ctx)}
	tests = append(tests, upgrade.VerifyPostInstallJobs(ctx, upgrade.VerifyPostJobsConfig{
		Namespace:    "knative-serving",
		FailOnNoJobs: failOnNoJobs,
	}))
	tests = append(tests, upgrade.VerifyPostInstallJobs(ctx, upgrade.VerifyPostJobsConfig{
		Namespace:    "knative-eventing",
		FailOnNoJobs: failOnNoJobs,
	}))
	tests = append(tests, eventingupgrade.PostUpgradeTests()...)
	tests = append(tests,
		kafkabrokerupgrade.ChannelPostUpgradeTest(),
		kafkabrokerupgrade.SourcePostUpgradeTest(glob),
		kafkabrokerupgrade.BrokerPostUpgradeTest(),
		kafkabrokerupgrade.NamespacedBrokerPostUpgradeTest(),
		kafkabrokerupgrade.SinkPostUpgradeTest(),
	)
	tests = append(tests, servingupgrade.ServingPostUpgradeTests()...)
	return tests
}

func postDowngradeTests() []pkgupgrade.Operation {
	tests := servingupgrade.ServingPostDowngradeTests()
	tests = append(tests,
		servingupgrade.CRDStoredVersionPostUpgradeTest(), // Check if CRD Stored version check works with downgrades.
		eventingupgrade.PostDowngradeTest(),
		eventingupgrade.CRDPostUpgradeTest(), // Check if CRD Stored version check works with downgrades.
		kafkabrokerupgrade.ChannelPostDowngradeTest(),
		kafkabrokerupgrade.SourcePostDowngradeTest(glob),
		kafkabrokerupgrade.BrokerPostDowngradeTest(),
		kafkabrokerupgrade.SinkPostDowngradeTest(),
	)
	return tests
}

func waitForServicesReady(ctx *test.Context) pkgupgrade.Operation {
	return pkgupgrade.NewOperation("WaitForServicesReady", func(c pkgupgrade.Context) {
		if err := test.WaitForReadyServices(ctx, "serving-tests"); err != nil {
			c.T.Error("Knative services not ready: ", err)
		}
		// TODO: Check if we need to sleep 30 more seconds like in the previous bash scripts.
	})
}
