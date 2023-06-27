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
	"context"
	"log"
	"testing"
	"time"

	"knative.dev/eventing-kafka-broker/control-plane/pkg/kafka"
	kafkabrokerupgrade "knative.dev/eventing-kafka-broker/test/upgrade"
	"knative.dev/eventing-kafka-broker/test/upgrade/continual"
	eventingtest "knative.dev/eventing/test"
	testlib "knative.dev/eventing/test/lib"
	"knative.dev/eventing/test/rekt/features/channel"
	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/eventing/test/rekt/resources/subscription"
	eventingupgrade "knative.dev/eventing/test/upgrade"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
	pkgTest "knative.dev/pkg/test"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	servingupgrade "knative.dev/serving/test/upgrade"

	"github.com/openshift-knative/serverless-operator/test"
	kafkafeatures "github.com/openshift-knative/serverless-operator/test/extensione2erekt/features"
	"github.com/openshift-knative/serverless-operator/test/upgrade"
	"github.com/openshift-knative/serverless-operator/test/upgrade/installation"
	"knative.dev/eventing-kafka-broker/test/rekt/features"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
	"knative.dev/reconciler-test/pkg/manifest"
)

var global environment.GlobalEnvironment

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
				ServingContinualTests(ctx),
				ChannelContinualTests(ctx),
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
	var tests []pkgupgrade.Operation
	tests = append(tests, EventingPreUpgradeTests()...)
	tests = append(tests, EventingKafkaBrokerPreUpgradeTests()...)
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
	tests = append(tests, EventingPostUpgradeTests()...)
	tests = append(tests, EventingKafkaBrokerPostUpgradeTests()...)
	tests = append(tests, servingupgrade.ServingPostUpgradeTests()...)
	return tests
}

func postDowngradeTests() []pkgupgrade.Operation {
	tests := servingupgrade.ServingPostDowngradeTests()
	tests = append(tests, EventingPostDowngradeTests()...)
	tests = append(tests, EventingKafkaBrokerPostDowngradeTests()...)
	tests = append(tests,
		servingupgrade.CRDStoredVersionPostUpgradeTest(), // Check if CRD Stored version check works with downgrades.
		eventingupgrade.CRDPostUpgradeTest(),             // Check if CRD Stored version check works with downgrades.
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

func TestMain(m *testing.M) {
	eventingtest.InitializeEventingFlags()

	restConfig, err := pkgTest.Flags.ClientConfig.GetRESTConfig()
	if err != nil {
		log.Fatal("Error building client config: ", err)
	}

	// Getting the rest config explicitly and passing it further will prevent re-initializing the flagset
	// in NewStandardGlobalEnvironment(). The upgrade tests use knative.dev/pkg/test which initializes the
	// flagset as well.
	global = environment.NewStandardGlobalEnvironment(func(cfg environment.Configuration) environment.Configuration {
		cfg.Config = restConfig
		return cfg
	})

	testlib.ReuseNamespace = eventingtest.EventingFlags.ReuseNamespace
	eventingupgrade.RunMainTest(m)
}

func defaultEnvironment(t *testing.T) (context.Context, environment.Environment) {
	return global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(4*time.Second, 10*time.Minute),
		environment.Managed(t),
	)
}

func EventingPreUpgradeTests() []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		InMemoryChannelPreUpgradeTest(),
	}
}

func EventingPostUpgradeTests() []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		InMemoryChannelPostUpgradeTest(),
	}
}

func EventingPostDowngradeTests() []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		InMemoryChannelPostDowngradeTest(),
	}
}

func InMemoryChannelPreUpgradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("InMemoryChannelPreUpgradeTest", func(c pkgupgrade.Context) {
		inMemoryChannelTest(c.T)
	})
}

func InMemoryChannelPostUpgradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("InMemoryChannelPostUpgradeTest", func(c pkgupgrade.Context) {
		inMemoryChannelTest(c.T)
	})
}

func InMemoryChannelPostDowngradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("InMemoryChannelPostDowngradeTest", func(c pkgupgrade.Context) {
		inMemoryChannelTest(c.T)
	})
}

func inMemoryChannelTest(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		t.Skip("Enable when testing upgrades from 1.30 to 1.31")
	}

	createSubscriberFn := func(ref *duckv1.KReference, uri string) manifest.CfgFn {
		return subscription.WithSubscriber(ref, uri)
	}
	env.Test(ctx, t, channel.ChannelChain(1, createSubscriberFn))
}

func EventingKafkaBrokerPreUpgradeTests() []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		KafkaChannelPreUpgradeTest(),
		KafkaBrokerPreUpgradeTest(),
		KafkaSinkAndSourcePreUpgradeTest(),
	}
}

func EventingKafkaBrokerPostUpgradeTests() []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		KafkaChannelPostUpgradeTest(),
		KafkaBrokerPostUpgradeTest(),
		KafkaSinkAndSourcePostUpgradeTest(),
	}
}

func EventingKafkaBrokerPostDowngradeTests() []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		KafkaChannelPostDowngradeTest(),
		KafkaBrokerPostDowngradeTest(),
		KafkaSinkAndSourcePostDowngradeTest(),
	}
}

func KafkaChannelPreUpgradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("KafkaChannelPreUpgradeTest", func(c pkgupgrade.Context) {
		kafkaChannelTest(c.T)
	})
}

func KafkaChannelPostUpgradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("KafkaChannelPostUpgradeTest", func(c pkgupgrade.Context) {
		kafkaChannelTest(c.T)
	})
}

func KafkaChannelPostDowngradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("KafkaChannelPostDowngradeTest", func(c pkgupgrade.Context) {
		kafkaChannelTest(c.T)
	})
}

func kafkaChannelTest(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		t.Skip("Enable when testing upgrades from 1.30 to 1.31")
	}

	channel_impl.EnvCfg.ChannelGK = "KafkaChannel.messaging.knative.dev"
	channel_impl.EnvCfg.ChannelV = "v1beta1"

	createSubscriberFn := func(ref *duckv1.KReference, uri string) manifest.CfgFn {
		return subscription.WithSubscriber(ref, uri)
	}
	env.Test(ctx, t, channel.ChannelChain(1, createSubscriberFn))
}

func KafkaBrokerPreUpgradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("KafkaBrokerPreUpgradeTest", func(c pkgupgrade.Context) {
		kafkaBrokerTest(c.T)
		namespacedKafkaBrokerTest(c.T)
	})
}

func KafkaBrokerPostUpgradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("KafkaBrokerPostUpgradeTest", func(c pkgupgrade.Context) {
		kafkaBrokerTest(c.T)
		namespacedKafkaBrokerTest(c.T)
	})
}

func KafkaBrokerPostDowngradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("KafkaBrokerPostDowngradeTest", func(c pkgupgrade.Context) {
		kafkaBrokerTest(c.T)
		namespacedKafkaBrokerTest(c.T)
	})
}

func kafkaBrokerTest(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, kafkafeatures.BrokerSmokeTest(kafka.BrokerClass))
}

func namespacedKafkaBrokerTest(t *testing.T) {
	ctx, env := defaultEnvironment(t)

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		// With Istio this issue happens often.
		t.Skip("https://issues.redhat.com/browse/SRVKE-1424")
	}

	env.Test(ctx, t, kafkafeatures.BrokerSmokeTest(kafka.NamespacedBrokerClass))
}

func KafkaSinkAndSourcePreUpgradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("SinkSourcePreUpgradeTest",
		func(c pkgupgrade.Context) {
			kafkaSinkAndSourceTest(c.T)
		})
}

func KafkaSinkAndSourcePostUpgradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("SinkSourcePostUpgradeTest",
		func(c pkgupgrade.Context) {
			kafkaSinkAndSourceTest(c.T)
		})
}

func KafkaSinkAndSourcePostDowngradeTest() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("SinkSourcePostDowngradeTest",
		func(c pkgupgrade.Context) {
			kafkaSinkAndSourceTest(c.T)
		})
}

func kafkaSinkAndSourceTest(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, features.KafkaSourceStructuredEvent())
	env.Test(ctx, t, features.KafkaSourceBinaryEvent())
}

func ChannelContinualTests(testCtx *test.Context) []pkgupgrade.BackgroundOperation {
	ctx, _ := defaultEnvironment(testCtx.T)

	// TODO: Enable when testing upgrades from 1.30 to 1.31
	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		return nil
	}

	return []pkgupgrade.BackgroundOperation{
		continual.ChannelTest(continual.ChannelTestOptions{}),
		continual.BrokerBackedByChannelTest(continual.ChannelTestOptions{}),
	}
}

func ServingContinualTests(testCtx *test.Context) []pkgupgrade.BackgroundOperation {
	ctx, _ := defaultEnvironment(testCtx.T)

	// https://issues.redhat.com/browse/SRVKS-1080
	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		return nil
	}

	return []pkgupgrade.BackgroundOperation{
		servingupgrade.ProbeTest(),
		servingupgrade.AutoscaleSustainingWithTBCTest(),
		servingupgrade.AutoscaleSustainingTest(),
	}
}
