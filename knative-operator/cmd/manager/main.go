package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeeventing"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativekafka"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeserving"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapr "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
	log                       = logf.Log.WithName("cmd")
)

func init() {
	prodConf := zap.NewProductionEncoderConfig()
	prodConf.EncodeTime = zapcore.ISO8601TimeEncoder
	logf.SetLogger(zapr.New(zapr.Encoder(zapcore.NewJSONEncoder(prodConf))))
}

func main() {
	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

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

	// Setup all Webhooks
	hookServer := mgr.GetWebhookServer()
	hookServer.Port = 9876
	hookServer.CertDir = "/apiserver.local.config/certificates"
	hookServer.KeyName = "apiserver.key"
	hookServer.CertName = "apiserver.crt"

	// Serving Webhooks
	hookServer.Register("/mutate-knativeservings", &webhook.Admission{Handler: &knativeserving.Configurator{}})
	hookServer.Register("/validate-knativeservings", &webhook.Admission{Handler: &knativeserving.Validator{}})
	// Eventing Webhooks
	hookServer.Register("/mutate-knativeeventings", &webhook.Admission{Handler: &knativeeventing.Configurator{}})
	hookServer.Register("/validate-knativeeventings", &webhook.Admission{Handler: &knativeeventing.Validator{}})
	// Kafka Webhooks
	hookServer.Register("/validate-knativekafkas", &webhook.Admission{Handler: &knativekafka.Validator{}})

	if err := setupMonitoring(cfg); err != nil {
		log.Error(err, "Failed to start monitoring")
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func setupMonitoring(cfg *rest.Config) error {
	cl, err := client.New(cfg, client.Options{})
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

	if err := common.SetupServerlessOperatorServiceMonitor(cfg, cl, metricsPort, metricsHost, operatorMetricsPort); err != nil {
		return fmt.Errorf("failed to setup the Service monitor: %w", err)
	}

	if err := common.InstallHealthDashboard(cl); err != nil {
		return fmt.Errorf("failed to setup the Knative Health Status Dashboard: %w", err)
	}
	return nil
}
