package servicemesh

import (
	"log"
	"os"
	"testing"

	"knative.dev/eventing/test/rekt/resources/broker"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/reconciler-test/pkg/environment"
)

var global environment.GlobalEnvironment

// TestMain is the first entry point for `go test`.
func TestMain(m *testing.M) {
	broker.EnvCfg.BrokerClass = "MTChannelBasedBroker"

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
