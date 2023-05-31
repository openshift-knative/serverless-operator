package extensione2erekt

import (
	"context"
	"os"
	"testing"

	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

var global environment.GlobalEnvironment

// TestMain is the first entry point for `go test`.
func TestMain(m *testing.M) {
	channel_impl.EnvCfg.ChannelGK = "KafkaChannel.messaging.knative.dev"
	channel_impl.EnvCfg.ChannelV = "v1beta1"

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
		environment.Managed(t),
	)
}
