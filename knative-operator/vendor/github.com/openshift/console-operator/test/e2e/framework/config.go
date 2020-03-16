package framework

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// GetConfig creates a *rest.Config for talking to a Kubernetes apiserver.
// Otherwise will assume running in cluster and use the cluster provided kubeconfig.
//
// Config precedence
//
// * KUBECONFIG environment variable pointing at a file
//
// * In-cluster config if running in cluster
//
// * $HOME/.kube/config if exists
//
// Note that if you have done `oc login...` the `current-context` will be used.
// this means that the test will run as whatever user is currently logged in.
// kube:admin has a token, but no certs.
// - kube:admin is an oauth user, this is why it has tokens.
// system:admin has certs, but no token.
// - system:admin is a cert based user only.
// other users, your mileage may vary.
func GetConfig() (*rest.Config, error) {
	// If an env variable is specified with the config location, use that
	if len(os.Getenv("KUBECONFIG")) > 0 {
		return clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	}
	// If no explicit location, try the in-cluster config
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory
	if usr, err := user.Current(); err == nil {
		if c, err := clientcmd.BuildConfigFromFlags(
			"", filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("could not locate a kubeconfig")
}

// authn operator approach to getting config.
// choose func as needed.
// https://github.com/openshift/cluster-authentication-operator/blob/master/test/library/client.go
// NewClientConfigForTest returns a config configured to connect to the api server
func NewClientConfigForTest() (*rest.Config, error) {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, &clientcmd.ConfigOverrides{ClusterInfo: api.Cluster{InsecureSkipTLSVerify: true}})
	config, err := clientConfig.ClientConfig()
	if err == nil {
		fmt.Printf("Found configuration for host %v.\n", config.Host)
	}
	return config, err
}
