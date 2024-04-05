package soak

import (
	"context"
	"flag"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/pkg/system"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

// var global environment.GlobalEnvironment

type SoakFlags struct {
	Duration     time.Duration
	Copies       int
	PollInterval time.Duration
	PollDuration time.Duration
}

var Flags SoakFlags

var global environment.GlobalEnvironment

// TestMain is the first entry point for `go test`.
func TestMain(m *testing.M) {
	channel_impl.EnvCfg.ChannelGK = "KafkaChannel.messaging.knative.dev"
	channel_impl.EnvCfg.ChannelV = "v1beta1"

	flag.DurationVar(&Flags.Duration, "soak-duration", 1*time.Hour, "Soak test duration")
	flag.IntVar(&Flags.Copies, "soak-copies", 1, "Number of copies for each soak test scenario")
	flag.DurationVar(&Flags.PollInterval, "soak-poll-interval", 5*time.Second, "Poll interval used in soak tests")
	flag.DurationVar(&Flags.PollDuration, "soak-poll-duration", 10*time.Minute, "Poll duration used in soak tests")

	restConfig, err := pkgTest.Flags.ClientConfig.GetRESTConfig()
	if err != nil {
		log.Fatal("Error building client config: ", err)
	}

	// Getting the rest config explicitly and passing it further will prevent re-initializing the flagset
	// in NewStandardGlobalEnvironment().
	global = environment.NewStandardGlobalEnvironment(func(cfg environment.Configuration) environment.Configuration {
		cfg.Config = restConfig
		return cfg
	})

	// Run the tests.
	os.Exit(m.Run())
}

func soakTestEnvironment(t *testing.T, namespace string) (context.Context, environment.Environment) {
	return global.Environment(
		environment.WithNamespace(namespace),
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		// Enables KnativeService in the scenario.
		//eventshub.WithKnativeServiceForwarder,
		environment.WithPollTimings(Flags.PollInterval, Flags.PollDuration),
		environment.WithTestLogger(t),
	)
}

type SoakFn func(context.Context, environment.Environment, *testing.T)

type SoakTest struct {
	/*
		Prefix to be used for namespaces. Actual test namespace is <prefix><copyId>
	*/
	NamespacePrefix string
	/*
		Function invoked during setup for each soak test copy.
		Use env.Test(ctx, t, f)  inside to handle cleanup of the resources created by the features.
	*/
	SetupFn SoakFn
	/*
		Function invoked for each iteration.
		Use env.Test(ctx, t, featureFn(SoakEnvFromContext(ctx)))  inside to dynamically create feature based on SoakEnv copy and iteration.
		Soak env does handle cleanup of all resources created by the features during the iteration
	*/
	IterationFn SoakFn
	/*
		Function invoked after the iterations complete. Could be used to teardown or state verification after the end of the soak test.
	*/
	TeardownFn SoakFn
}

type soakKey struct{}

type SoakEnv interface {
	CopyID() int
	Iteration() int
	Namespace() string
}

type soakEnvImpl struct {
	copyID    int
	iteration int
	namespace string
}

func (env *soakEnvImpl) CopyID() int {
	return env.copyID
}

func (env *soakEnvImpl) Iteration() int {
	return env.iteration
}

func (env *soakEnvImpl) Namespace() string {
	return env.namespace
}

func soakEnvImplFromContext(ctx context.Context) *soakEnvImpl {
	if se, ok := ctx.Value(soakKey{}).(*soakEnvImpl); ok {
		return se
	}
	panic("no soak environment found in the context, make sure you're executing a soak test code within a RunSoakTest")
}

func SoakEnvFromContext(ctx context.Context) SoakEnv {
	return soakEnvImplFromContext(ctx)
}

func RunSoakTest(t *testing.T, test SoakTest, copies int) {
	for copyID := 0; copyID < copies; copyID++ {
		copyID := copyID
		namespace := test.NamespacePrefix + strconv.Itoa(copyID)
		t.Run(namespace, func(t *testing.T) {
			t.Parallel()

			since := time.Now()

			ctx, env := soakTestEnvironment(t, namespace)

			// Execute the setup "features", store the references created during setup for cleanup at the end
			setupCtx := context.WithValue(ctx, soakKey{}, &soakEnvImpl{
				copyID:    copyID,
				iteration: -1,
				namespace: namespace,
			})

			if test.SetupFn != nil {
				test.SetupFn(setupCtx, env, t)
			}
			if t.Failed() {
				feature.LogReferences(env.References()...)(ctx, t)
				return
			}

			iteration := 0
			// Repeat the soak test for the duration specified by --soak-duration flag
			for since.Add(Flags.Duration).After(time.Now()) {
				// During each iteration, generate the "iteration" features and run them as Tests
				// Cleanup all resources for these features at the end of the iteration

				// For each iteration, we create a new empty Environment, while keeping the same Context
				// This way we can delete all resources created during an iteration, but we can still use per-soak test
				// context, allowing us, for example, an iteration-scoped eventshub sender sending events to
				// a test-scoped eventshub receiver
				_, iterationEnv := global.Environment(
					environment.WithNamespace(namespace),
				)

				iterationCtx := context.WithValue(ctx, soakKey{}, &soakEnvImpl{
					copyID:    copyID,
					iteration: iteration,
					namespace: namespace,
				})

				iterationCtx = environment.ContextWith(iterationCtx, iterationEnv)

				if test.IterationFn != nil {
					test.IterationFn(iterationCtx, iterationEnv, t)
				}

				if t.Failed() {
					feature.LogReferences(env.References()...)(ctx, t)
					feature.LogReferences(iterationEnv.References()...)(ctx, t)
					return
				}

				// Cleanup all resources created in this iteration
				err := feature.DeleteResources(ctx, t, iterationEnv.References())
				if err != nil {
					feature.LogReferences(env.References()...)(ctx, t)
					feature.LogReferences(iterationEnv.References()...)(ctx, t)
					t.Fatalf("error deleting resources: %v", err)
				}

				iteration++
			}

			if iteration == 0 {
				t.Errorf("No iteration ran")
			}

			teardownCtx := context.WithValue(ctx, soakKey{}, &soakEnvImpl{
				copyID:    copyID,
				iteration: iteration,
				namespace: namespace,
			})

			if test.TeardownFn != nil {
				test.TeardownFn(teardownCtx, env, t)
			}

			if t.Failed() {
				feature.LogReferences(env.References()...)(ctx, t)
				return
			}

			// cleanup all the references from the setup phase
			err := feature.DeleteResources(ctx, t, env.References())
			if err != nil {
				feature.LogReferences(env.References()...)(ctx, t)
				t.Fatalf("error deleting resources: %v", err)
			}
			env.Finish()
		})
	}
}

func RunSoakTestWithDefaultCopies(t *testing.T, test SoakTest) {
	RunSoakTest(t, test, Flags.Copies)
}
