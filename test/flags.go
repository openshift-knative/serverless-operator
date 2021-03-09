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
	Kubeconfigs     string // Path to .kube/config
	CatalogSource   string // CatalogSource in the openshift-marketplace namespace for the Serverless operator Subscription
	Channel         string // Serverless operator Subscription channel
	Subscription    string // Serverless operator Subscription name
	UpgradeChannel  string // Target OLM channel for upgrades
	CSV             string // Target CSV for upgrades
	ServingVersion  string // Target Serving version for upgrades
	EventingVersion string // Target Eventing version for upgrades
	OpenShiftImage  string // Target OpenShift image for upgrades
}

func initializeFlags() *FlagsStruct {
	var f FlagsStruct

	var defaultKubeconfig string
	if usr, err := user.Current(); err == nil {
		defaultKubeconfig = path.Join(usr.HomeDir, ".kube/config")
	}
	flag.StringVar(&f.Kubeconfigs, "kubeconfigs", defaultKubeconfig,
		"Provide the path to the `kubeconfig` file you'd like to use for these tests. The `current-context` will be used.")
	flag.StringVar(&f.CatalogSource, "catalogsource", "serverless-operator",
		"CatalogSource in the openshift-marketplace namespace for the Serverless operator Subscription, \"serverless-operator\" by default")
	flag.StringVar(&f.Channel, "channel", "",
		"Serverless operator Subscription channel, empty by default.")
	flag.StringVar(&f.Subscription, "subscription", "serverless-operator",
		"Serverless operator Subscription name, \"serverless-operator\" by default.")
	flag.StringVar(&f.UpgradeChannel, "upgradechannel", "",
		"OLM channel to be used during upgrades, empty by default.")
	flag.StringVar(&f.CSV, "csv", "",
		"Target ClusterServiceVersion for upgrade tests, empty by default.")
	flag.StringVar(&f.ServingVersion, "servingversion", "",
		"Target Serving version for upgrade tests, empty by default.")
	flag.StringVar(&f.EventingVersion, "eventingversion", "",
		"Target Eventing version for upgrade tests, empty by default.")
	flag.StringVar(&f.OpenShiftImage, "openshiftimage", "",
		"Target OpenShift image for cluster upgrades, empty by default.")

	return &f
}

// Main is a main test runner
func Main(m *testing.M) {
	// go1.13+ testing flags regression fix: https://github.com/golang/go/issues/31859
	flag.Parse()
	os.Exit(m.Run())
}
