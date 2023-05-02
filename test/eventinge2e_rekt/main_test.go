package eventinge2e_rekt

import (
	"os"
	"testing"

	"knative.dev/reconciler-test/pkg/environment"
)

var global environment.GlobalEnvironment

// TestMain is the first entry point for `go test`.
func TestMain(m *testing.M) {
	global = environment.NewStandardGlobalEnvironment()

	// Run the tests.
	os.Exit(m.Run())
}
