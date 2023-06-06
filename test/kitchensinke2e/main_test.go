package kitchensinke2e

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"

	// Make sure to initialize flags from knative.dev/pkg/test before parsing them.
	pkgTest "knative.dev/pkg/test"

	"knative.dev/reconciler-test/pkg/environment"
)

var global environment.GlobalEnvironment

func defaultContext(t *testing.T) (context.Context, environment.Environment) {
	return global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(4*time.Second, 15*time.Minute),
		environment.Managed(t),
	)
}

// TestMain is the first entry point for `go test`.
func TestMain(m *testing.M) {
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
