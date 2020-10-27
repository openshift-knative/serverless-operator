package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	"github.com/spf13/pflag"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)
var log = logf.Log.WithName("cmd")

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Set("zap-time-encoding", "iso8601")

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()
	// Become the leader before proceeding
	// This needs to remain "knative-serving-openshift-lock" to allow for safe upgrades.
	err = leader.Become(ctx, "knative-serving-openshift-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          "", // The serverless operator always watches all namespaces.
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all webhooks
	if err := webhook.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := setupMonitoring(ctx, cfg); err != nil {
		log.Error(err, "Failed to start monitoring")
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func setupMonitoring(ctx context.Context, cfg *rest.Config) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to create cluster config: %w", err)
	}

	cl, err := client.New(config, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to create a client: %w", err)
	}

	namespace := os.Getenv(common.NamespaceEnvKey)
	if namespace == "" {
		return errors.New("NAMESPACE not provided via environment")
	}

	operatorDeployment, err := common.GetServerlessOperatorDeployment(cl, namespace)
	if err != nil {
		return err
	}

	if err = common.SetupMonitoringRequirements(cl, operatorDeployment); err != nil {
		return fmt.Errorf("failed to setup monitoring resources: %w", err)
	}

	if err := common.SetupServerlessOperatorServiceMonitor(ctx, cfg, cl, metricsPort, metricsHost, operatorMetricsPort); err != nil {
		return fmt.Errorf("failed to setup the Service monitor: %w", err)
	}

	if err := common.InstallHealthDashboard(cl); err != nil {
		return fmt.Errorf("failed to setup the Knative Health Status Dashboard: %w", err)
	}
	return nil
}
