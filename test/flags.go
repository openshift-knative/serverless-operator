package test

import (
	"errors"
	"flag"
	"os"
	"os/user"
	"path"
	"testing"

	upstreamtest "knative.dev/pkg/test"
)

// Flags holds the initialized test flags
var Flags = initializeFlags(upstreamtest.Flags)

// FlagsStruct is struct that defines testing options
type FlagsStruct struct {
	Kubeconfigs string // Path to .kube/config
}

func initializeFlags(upstreamFlags *upstreamtest.EnvironmentFlags) *FlagsStruct {
	var f FlagsStruct

	var defaultKubeconfig string
	if usr, err := user.Current(); err == nil {
		defaultKubeconfig = path.Join(usr.HomeDir, ".kube/config")
	}
	flag.StringVar(&f.Kubeconfigs, "kubeconfigs", defaultKubeconfig,
		"Provide the path to the `kubeconfig` file you'd like to use for these tests. The `current-context` will be used.")
	if upstreamFlags == nil {
		panic(errors.New("upstream flags must be defined first"))
	}

	return &f
}

// Main is a main test runner
func Main(m *testing.M) {
	// go1.13+ testing flags regression fix: https://github.com/golang/go/issues/31859
	flag.Parse()
	os.Exit(m.Run())
}
