package soake2erekt

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/injection/clients/dynamicclient"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/pkg/system"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

// var global environment.GlobalEnvironment

type SoakFlagsStruct struct {
	Duration     time.Duration
	Copies       int
	PollInterval time.Duration
	PollDuration time.Duration
}

var Flags SoakFlagsStruct

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
		environment.InNamespace(namespace),
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

/*
SoakFeatureFn represents part of the soak test to be run repeatedly, over a number of copies in parallel
The features are generated dynamically by a function, so that it is possible to generate unique resource names
for each test copy, or in each iteration
*/
type SoakFeatureFn func(SoakEnv) *feature.Feature

type SoakTest struct {
	/*
		Prefix to be used for namespaces. Actual test namespace is <prefix><copyId>
	*/
	NamespacePrefix string
	/*
		Function invoked during setup for each soak test copy. Use RunSoakFeature* inside to handle cleanup of the resources created by the features
	*/
	SetupFn SoakFn
	/*
		Function invoked for each iteration.
		Use RunSoakFeature* inside to handle cleanup of the resources created by the features.
		Use RunSoakFeatureFn* inside to also handle creation of the features dynamically based on the context of the soak test
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

				// TODO: can we do this? (use an empty env for the iteration, but keep using the original context?)
				_, iterationEnv := global.Environment(
					environment.InNamespace(namespace),
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
				err := deleteResources(ctx, t, iterationEnv.References())
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
			err := deleteResources(ctx, t, env.References())
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

func RunSoakFeature(ctx context.Context, env environment.Environment, t *testing.T, f *feature.Feature) {
	env.Test(ctx, t, f)
}

func RunSoakFeatureFn(ctx context.Context, env environment.Environment, t *testing.T, sfn SoakFeatureFn) {
	soakEnv := soakEnvImplFromContext(ctx)
	RunSoakFeature(ctx, env, t, sfn(soakEnv))
}

func RunSoakFeatureFnWithMapping[X any](ctx context.Context, env environment.Environment, t *testing.T, sfn func(X) *feature.Feature, mf func(SoakEnv) X) {
	soakEnv := soakEnvImplFromContext(ctx)
	f := sfn(mf(soakEnv))
	RunSoakFeature(ctx, env, t, f)
}

/*
copy from features, with Poll changes to use the ones provided by environment.PollTimingsFromContext
TODO: move upstream
*/
func deleteResources(ctx context.Context, t *testing.T, refs []corev1.ObjectReference) error {
	dc := dynamicclient.Get(ctx)

	for _, ref := range refs {

		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return fmt.Errorf("could not parse GroupVersion for %+v", ref.APIVersion)
		}

		resource := apis.KindToResource(gv.WithKind(ref.Kind))
		t.Logf("Deleting %s/%s of GVR: %+v", ref.Namespace, ref.Name, resource)

		deleteOptions := &metav1.DeleteOptions{}
		// Set delete propagation policy to foreground
		foregroundDeletePropagation := metav1.DeletePropagationForeground
		deleteOptions.PropagationPolicy = &foregroundDeletePropagation

		err = dc.Resource(resource).Namespace(ref.Namespace).Delete(ctx, ref.Name, *deleteOptions)
		// Ignore not found errors.
		if err != nil && !apierrors.IsNotFound(err) {
			t.Logf("Warning, failed to delete %s/%s of GVR: %+v: %v", ref.Namespace, ref.Name, resource, err)
		}
	}

	interval, duration := environment.PollTimingsFromContext(ctx)
	err := wait.Poll(interval, duration, func() (bool, error) {
		for _, ref := range refs {
			gv, err := schema.ParseGroupVersion(ref.APIVersion)
			if err != nil {
				return false, fmt.Errorf("could not parse GroupVersion for %+v", ref.APIVersion)
			}

			resource := apis.KindToResource(gv.WithKind(ref.Kind))
			t.Logf("Deleting %s/%s of GVR: %+v", ref.Namespace, ref.Name, resource)

			_, err = dc.Resource(resource).
				Namespace(ref.Namespace).
				Get(ctx, ref.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				feature.LogReferences(ref)(ctx, t)
				return false, fmt.Errorf("failed to get resource %+v %s/%s: %w", resource, ref.Namespace, ref.Name, err)
			}

			t.Logf("Resource %+v %s/%s still present", resource, ref.Namespace, ref.Name)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for resources to be deleted: %w", err)
	}

	return nil
}
