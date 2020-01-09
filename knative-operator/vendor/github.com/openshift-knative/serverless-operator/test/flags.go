package test

import (
	"flag"
	"os"
	"os/user"
	"path"
	"testing"
)

// Flags holds the initialized test flags
var Flags = initializeFlags()

// FlagsStruct is struct that defines testing options
type FlagsStruct struct {
	Kubeconfigs string // Path to .kube/config
}

func initializeFlags() *FlagsStruct {
	var f FlagsStruct

	var defaultKubeconfig string
	if usr, err := user.Current(); err == nil {
		defaultKubeconfig = path.Join(usr.HomeDir, ".kube/config")
	}
	flag.StringVar(&f.Kubeconfigs, "kubeconfigs", defaultKubeconfig,
		"Provide the path to the `kubeconfig` file you'd like to use for these tests. The `current-context` will be used.")

	return &f
}

// Main is a main test runner
func Main(m *testing.M) {
	// go1.13+ testing flags regression fix: https://github.com/golang/go/issues/31859
	flag.Parse()
	os.Exit(m.Run())
}
