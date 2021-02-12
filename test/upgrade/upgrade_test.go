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
	"github.com/openshift-knative/serverless-operator/test/upgrade/installation"
	"os"
	"testing"

	"go.uber.org/zap"
	_ "knative.dev/pkg/system/testing"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	servingupgrade "knative.dev/serving/test/upgrade"
	eventingupgrade "knative.dev/eventing/test/upgrade"
)

func TestServerlessUpgrade(t *testing.T) {
	c := newUpgradeConfig(t)
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade:    preUpgradeTests(),
			PostUpgrade:   postUpgradeTests(),
			Continual:     []pkgupgrade.BackgroundOperation{
				// TODO(mgencur): SRVKS-698 Investigate failing Autoscale tests.
				servingupgrade.ProbeTest(),
				eventingupgrade.ContinualTest(),
			},
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith: []pkgupgrade.Operation{ installation.UpgradeServerless() },
		},
	}
	suite.Execute(c)
}

func TestClusterUpgrade(t *testing.T) {
	if os.Getenv("UPGRADE_CLUSTER") != "true" {
		t.Skip("Cluster upgrade tests disabled unless UPGRADE_CLUSTER=true env var defined.")
	}
	c := newUpgradeConfig(t)
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade:    preUpgradeTests(),
			PostUpgrade:   postUpgradeTests(),
			// Do not include continual tests as they're failing across cluster upgrades.
		},
		Installations: pkgupgrade.Installations{
			UpgradeWith: []pkgupgrade.Operation{ installation.UpgradeCluster() },
		},
	}
	suite.Execute(c)
}

func preUpgradeTests() []pkgupgrade.Operation {
	tests := []pkgupgrade.Operation { eventingupgrade.PreUpgradeTest() }
	// We might want to skip pre-upgrade test if we want to re-use the services
	// from the previous run. For example, to let them survive both Serverless
	// and OCP upgrades. This allows for more variants of tests, with different
	// order of upgrades.
	if os.Getenv("SKIP_SERVING_PRE_UPGRADE") == "true" {
		return tests
	}
	return append(tests, servingupgrade.ServingPreUpgradeTests()...)
}

func postUpgradeTests() []pkgupgrade.Operation {
	tests := servingupgrade.ServingPostUpgradeTests()
	return append(tests, eventingupgrade.PostUpgradeTest())
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
