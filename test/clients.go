package test

import (
	"os"
	"os/signal"
	"strings"
	"testing"

	configV1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	consolev1 "github.com/openshift/client-go/console/clientset/versioned/typed/console/v1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client"
	olmversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	monclientv1 "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	eventingversioned "knative.dev/eventing/pkg/client/clientset/versioned"
	operatorversioned "knative.dev/operator/pkg/client/clientset/versioned"
	operatorv1alpha1 "knative.dev/operator/pkg/client/clientset/versioned/typed/operator/v1alpha1"
	servingversioned "knative.dev/serving/pkg/client/clientset/versioned"

	// Extensions
	kafkachannelversioned "knative.dev/eventing-contrib/kafka/channel/pkg/client/clientset/versioned"
	kafkasourceversioned "knative.dev/eventing-contrib/kafka/source/pkg/client/clientset/versioned"
)

// Context holds objects related to test execution
type Context struct {
	Name        string
	T           *testing.T
	Clients     *Clients
	CleanupList []CleanupFunc
}

// Clients holds instances of interfaces for making requests to various APIs
type Clients struct {
	Kube               *kubernetes.Clientset
	Operator           operatorv1alpha1.OperatorV1alpha1Interface
	Serving            *servingversioned.Clientset
	Eventing           *eventingversioned.Clientset
	OLM                olmversioned.Interface
	Dynamic            dynamic.Interface
	Config             *rest.Config
	Route              routev1.RouteV1Interface
	ProxyConfig        configV1.ConfigV1Interface
	ConsoleCLIDownload consolev1.ConsoleCLIDownloadInterface
	MonitoringClient   monclientv1.MonitoringV1Interface
	KafkaSource        *kafkasourceversioned.Clientset
	KafkaChannel       *kafkachannelversioned.Clientset
}

// CleanupFunc defines a function that is called when the respective resource
// should be deleted. When creating resources the user should also create a CleanupFunc
// and register with the Context
type CleanupFunc func() error

var clients []*Clients

// setupClientsOnce creates Clients for all kubeconfigs passed from the command line
func setupClientsOnce(t *testing.T) {
	if len(clients) == 0 {
		kubeconfigs := strings.Split(Flags.Kubeconfigs, ",")
		for _, cfg := range kubeconfigs {
			clientset, err := NewClients(cfg)
			if err != nil {
				t.Fatalf("Couldn't initialize clients for config %s: %v", cfg, err)
			}
			clients = append(clients, clientset)
		}
	}
}

// SetupClusterAdmin returns context for Cluster Admin user
func SetupClusterAdmin(t *testing.T) *Context {
	setupClientsOnce(t)
	return contextAtIndex(0, "ClusterAdmin", t)
}

// SetupProjectAdmin returns context for Project Admin user
func SetupProjectAdmin(t *testing.T) *Context {
	setupClientsOnce(t)
	return contextAtIndex(1, "ProjectAdmin", t)
}

// SetupEdit returns context for user with Edit role
func SetupEdit(t *testing.T) *Context {
	setupClientsOnce(t)
	return contextAtIndex(2, "Edit", t)
}

// SetupView returns context for user with View role
func SetupView(t *testing.T) *Context {
	setupClientsOnce(t)
	return contextAtIndex(3, "View", t)
}

func contextAtIndex(i int, role string, t *testing.T) *Context {
	if len(clients) < i+1 {
		t.Fatalf("kubeconfig for user with %s role not present", role)
	}

	return &Context{
		Name:    role,
		T:       t,
		Clients: clients[i],
	}
}

// NewClients instantiates and returns several clientsets required for making request to the
// Knative cluster
func NewClients(kubeconfig string) (*Clients, error) {
	clients := &Clients{}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// We poll, so set our limits high.
	cfg.QPS = 100
	cfg.Burst = 200

	clients.Config = cfg
	clients.Kube = kubernetes.NewForConfigOrDie(cfg)
	clients.Dynamic = dynamic.NewForConfigOrDie(cfg)
	clients.Operator = operatorversioned.NewForConfigOrDie(cfg).OperatorV1alpha1()
	clients.Serving = servingversioned.NewForConfigOrDie(cfg)
	clients.Eventing = eventingversioned.NewForConfigOrDie(cfg)
	clients.Route = routev1.NewForConfigOrDie(cfg)
	clients.ProxyConfig = configV1.NewForConfigOrDie(cfg)
	clients.KafkaSource = kafkasourceversioned.NewForConfigOrDie(cfg)
	clients.KafkaChannel = kafkachannelversioned.NewForConfigOrDie(cfg)

	clients.OLM, err = client.NewClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	clients.ConsoleCLIDownload = consolev1.NewForConfigOrDie(cfg).ConsoleCLIDownloads()
	if err != nil {
		return nil, err
	}

	clients.MonitoringClient = monclientv1.NewForConfigOrDie(cfg)

	return clients, nil
}

// Cleanup for all contexts
func CleanupAll(t *testing.T, contexts ...*Context) {
	for _, ctx := range contexts {
		ctx.Cleanup(t)
	}
}

// Cleanup iterates through the list of registered CleanupFunc functions and calls them
func (ctx *Context) Cleanup(t *testing.T) {
	for _, f := range ctx.CleanupList {
		if err := f(); err != nil {
			t.Logf("Failed to clean up: %v", err)
		}
	}
}

// AddToCleanup adds the cleanup function as the first function to the cleanup list,
// we want to delete the last thing first
func (ctx *Context) AddToCleanup(f CleanupFunc) {
	ctx.CleanupList = append([]CleanupFunc{f}, ctx.CleanupList...)
}

// CleanupOnInterrupt will execute the function cleanup if an interrupt signal is caught
func CleanupOnInterrupt(t *testing.T, cleanup func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			t.Logf("Test interrupted, cleaning up.")
			cleanup()
			os.Exit(1)
		}
	}()
}
