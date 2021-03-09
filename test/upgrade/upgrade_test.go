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

package upgrade

import (
	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/upgrade/installation"
	"os"
	"testing"

	"go.uber.org/zap"
	eventingupgrade "knative.dev/eventing/test/upgrade"
	_ "knative.dev/pkg/system/testing"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	servingupgrade "knative.dev/serving/test/upgrade"
)

func TestServerlessUpgrade(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
	cfg := newUpgradeConfig(t)
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade:  preUpgradeTests(),
			PostUpgrade: postUpgradeTests(ctx),
			Continual: []pkgupgrade.BackgroundOperation{
				// TODO: SRVKS-698 Investigate AutoscaleSustainingWithTBCTest flakiness and re-enable.
				servingupgrade.ProbeTest(),
				servingupgrade.AutoscaleSustainingTest(),
				eventingupgrade.ContinualTest(),
			},
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith: []pkgupgrade.Operation{
				pkgupgrade.NewOperation("UpgradeServerless", func(c pkgupgrade.Context) {
					if err := installation.UpgradeServerless(ctx); err != nil {
						c.T.Error("Serverless upgrade failed:", err)
					}
				}),
			},
		},
	}
	suite.Execute(cfg)
}

func TestClusterUpgrade(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	if os.Getenv("UPGRADE_CLUSTER") != "true" {
		t.Skip("Cluster upgrade tests disabled unless UPGRADE_CLUSTER=true env var defined.")
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

func preUpgradeTests() []pkgupgrade.Operation {
	tests := []pkgupgrade.Operation{eventingupgrade.PreUpgradeTest()}
	// We might want to skip pre-upgrade test if we want to re-use the services
	// from the previous run. For example, to let them survive both Serverless
	// and OCP upgrades. This allows for more variants of tests, with different
	// order of upgrades.
	if os.Getenv("SKIP_SERVING_PRE_UPGRADE") == "true" {
		return tests
	}
	return append(tests, servingupgrade.ServingPreUpgradeTests()...)
}

func postUpgradeTests(ctx *test.Context) []pkgupgrade.Operation {
	var tests []pkgupgrade.Operation
	tests = append(tests, waitForServicesReady(ctx))
	tests = append(tests, servingupgrade.ServingPostUpgradeTests()...)
	tests = append(tests, eventingupgrade.PostUpgradeTest())
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
