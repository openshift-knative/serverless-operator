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

	"go.uber.org/zap"
	kafkabrokerupgrade "knative.dev/eventing-kafka-broker/test/upgrade"
	kafkaupgrade "knative.dev/eventing-kafka/test/upgrade"
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

func TestServerlessUpgrade(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
	cfg := newUpgradeConfig(t)
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			//PreUpgrade:    preUpgradeTests(),
			//PostUpgrade:   postUpgradeTests(ctx),
			PostDowngrade: postDowngradeTests(),
			Continual: merge(
				[]pkgupgrade.BackgroundOperation{
					servingupgrade.ProbeTest(),
					servingupgrade.AutoscaleSustainingWithTBCTest(),
					servingupgrade.AutoscaleSustainingTest(),
				},
				//kafkaupgrade.ChannelContinualTests(continual.ChannelTestOptions{}),
				//kafkabrokerupgrade.BrokerContinualTests(),
				//kafkabrokerupgrade.SinkContinualTests(),
			),
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith: []pkgupgrade.Operation{
				pkgupgrade.NewOperation("UpgradeServerless", func(c pkgupgrade.Context) {
					if err := installation.UpgradeServerless(ctx); err != nil {
						c.T.Error("Serverless upgrade failed:", err)
					}
				}),
			},
			DowngradeWith: downgrade(ctx),
		},
	}
	suite.Execute(cfg)
}

func TestClusterUpgrade(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	if !test.Flags.UpgradeOpenShift {
		t.Skip("Cluster upgrade tests disabled unless enabled by a flag.")
	}
	cfg := newUpgradeConfig(t)
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade:  preUpgradeTests(),
			PostUpgrade: postUpgradeTests(ctx),
			// Do not include continual tests as they're failing across cluster upgrades.
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith: []pkgupgrade.Operation{
				pkgupgrade.NewOperation("OpenShift Upgrade", func(c pkgupgrade.Context) {
					if err := installation.UpgradeOpenShift(ctx); err != nil {
						c.T.Error("OpenShift upgrade failed:", err)
					}
				}),
			},
		},
	}
	suite.Execute(cfg)
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
		kafkaupgrade.ChannelPreUpgradeTest(),
		kafkaupgrade.SourcePreUpgradeTest(),
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

func postUpgradeTests(ctx *test.Context) []pkgupgrade.Operation {
	tests := []pkgupgrade.Operation{waitForServicesReady(ctx)}
	tests = append(tests, upgrade.VerifyPostInstallJobs(ctx, upgrade.VerifyPostJobsConfig{
		Namespace:    "knative-serving",
		FailOnNoJobs: true,
	}))
	tests = append(tests, upgrade.VerifyPostInstallJobs(ctx, upgrade.VerifyPostJobsConfig{
		Namespace:    "knative-eventing",
		FailOnNoJobs: true,
	}))
	//tests = append(tests, eventingupgrade.PostUpgradeTests()...)
	//tests = append(tests,
	//	kafkaupgrade.ChannelPostUpgradeTest(),
	//	kafkaupgrade.SourcePostUpgradeTest(),
	//	kafkabrokerupgrade.BrokerPostUpgradeTest(),
	//	kafkabrokerupgrade.SinkPostUpgradeTest(),
	//	upgrade.VerifySugarControllerDeletion(ctx),
	//)
	//tests = append(tests, servingupgrade.ServingPostUpgradeTests()...)
	return tests
}

func postDowngradeTests() []pkgupgrade.Operation {
	if test.Flags.SkipDowngrade {
		return nil
	}
	tests := servingupgrade.ServingPostDowngradeTests()
	tests = append(tests,
		servingupgrade.CRDStoredVersionPostUpgradeTest(), // Check if CRD Stored version check works with downgrades.
		eventingupgrade.PostDowngradeTest(),
		eventingupgrade.CRDPostUpgradeTest(), // Check if CRD Stored version check works with downgrades.
		kafkaupgrade.ChannelPostDowngradeTest(),
		kafkaupgrade.SourcePostDowngradeTest(),
		kafkabrokerupgrade.BrokerPostDowngradeTest(),
		kafkabrokerupgrade.SinkPostDowngradeTest(),
	)
	return tests
}

func downgrade(ctx *test.Context) []pkgupgrade.Operation {
	if test.Flags.SkipDowngrade {
		return nil
	}
	return []pkgupgrade.Operation{
		pkgupgrade.NewOperation("DowngradeServerless", func(c pkgupgrade.Context) {
			if err := installation.DowngradeServerless(ctx); err != nil {
				c.T.Error("Serverless downgrade failed:", err)
			}
		}),
	}
}

func waitForServicesReady(ctx *test.Context) pkgupgrade.Operation {
	return pkgupgrade.NewOperation("WaitForServicesReady", func(c pkgupgrade.Context) {
		if err := test.WaitForReadyServices(ctx, "serving-tests"); err != nil {
			c.T.Error("Knative services not ready: ", err)
		}
		// TODO: Check if we need to sleep 30 more seconds like in the previous bash scripts.
	})
}

func newUpgradeConfig(t *testing.T) pkgupgrade.Configuration {
	log, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	return pkgupgrade.Configuration{T: t, Log: log}
}

func TestMain(m *testing.M) {
	eventingupgrade.RunMainTest(m)
}
