package eventinge2erekt

import (
	"context"
	"os"
	"testing"
	"time"

	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

var global environment.GlobalEnvironment

// TestMain is the first entry point for `go test`.
func TestMain(m *testing.M) {
	broker.EnvCfg.BrokerClass = "MTChannelBasedBroker"

	global = environment.NewStandardGlobalEnvironment()

	// Run the tests.
	os.Exit(m.Run())
}

func defaultEnvironment(t *testing.T) (context.Context, environment.Environment) {
	return global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		// Enables KnativeService in the scenario.
		eventshub.WithKnativeServiceForwarder,
		environment.WithPollTimings(5*time.Second, 4*time.Minute),
		environment.Managed(t),
	)
}
