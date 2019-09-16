package test

import (
	"flag"
	"os/user"
	"path"
)

// Flags holds the initialized test flags
var Flags = initializeFlags()

type TestFlags struct {
	Kubeconfig string // Path to .kube/config
}

func initializeFlags() *TestFlags {
	var f TestFlags

	var defaultKubeconfig string
	if usr, err := user.Current(); err == nil {
		defaultKubeconfig = path.Join(usr.HomeDir, ".kube/config")
	}
	flag.StringVar(&f.Kubeconfig, "kubeconfig", defaultKubeconfig,
		"Provide the path to the `kubeconfig` file you'd like to use for these tests. The `current-context` will be used.")

	flag.Parse()

	return &f
}
