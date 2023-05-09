package extensione2erekt

import (
	"os"
	"testing"

	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/reconciler-test/pkg/environment"
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
