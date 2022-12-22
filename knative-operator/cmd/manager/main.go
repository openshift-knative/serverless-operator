package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/dashboards/health"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeeventing"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativekafka"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeserving"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/rest"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/consoleutil"
	configv1 "github.com/openshift/api/config/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapr "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
	healthPort  int32 = 8687
	log               = logf.Log.WithName("cmd")
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

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:              "", // The serverless operator always watches all namespaces.
		LeaderElection:         true,
		LeaderElectionID:       "knative-serving-openshift-lock",
		MetricsBindAddress:     fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		HealthProbeBindAddress: fmt.Sprintf(":%d", healthPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Add readiness probe
	if err := mgr.AddReadyzCheck("ready-ping", healthz.Ping); err != nil {
		log.Error(err, "unable to add a readiness check")
		os.Exit(1)
	}

	// Add liveness probe
	if err := mgr.AddHealthzCheck("health-ping", healthz.Ping); err != nil {
		log.Error(err, "unable to add a health check")
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

	decoder, err := admission.NewDecoder(mgr.GetScheme())
	if err != nil {
		log.Error(err, "failed to create decoder")
		os.Exit(1)
	}

	// Serving Webhooks
	hookServer.Register("/mutate-knativeservings", &webhook.Admission{Handler: knativeserving.NewConfigurator(decoder)})
	hookServer.Register("/validate-knativeservings", &webhook.Admission{Handler: knativeserving.NewValidator(mgr.GetClient(), decoder)})
	// Eventing Webhooks
	hookServer.Register("/mutate-knativeeventings", &webhook.Admission{Handler: knativeeventing.NewConfigurator(decoder)})
	hookServer.Register("/validate-knativeeventings", &webhook.Admission{Handler: knativeeventing.NewValidator(mgr.GetClient(), decoder)})
	// Kafka Webhooks
	hookServer.Register("/mutate-knativekafkas", &webhook.Admission{Handler: knativekafka.NewConfigurator(decoder)})
	hookServer.Register("/validate-knativekafkas", &webhook.Admission{Handler: knativekafka.NewValidator(mgr.GetClient(), decoder)})

	if err := setupServerlesOperatorMonitoring(cfg); err != nil {
		log.Error(err, "Failed to start monitoring")
	}

	log.Info("Starting the Cmd.")

	go func() {
		// This web server is unimportant enough to not bother connecting its lifecycle to
		// signal handling so the process can just tear it down with it.
		log.Info("Serving CLI artifacts on :8080")
		http.Handle("/", http.FileServer(http.Dir("/cli-artifacts")))
		server := http.Server{Addr: ":8080", ReadHeaderTimeout: time.Minute}
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(err, "Failed to launch CLI artifact server")
		}
	}()

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func setupServerlesOperatorMonitoring(cfg *rest.Config) error {
	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to create a client: %w", err)
	}
	namespace := os.Getenv(common.NamespaceEnvKey)
	if namespace == "" {
		return errors.New("NAMESPACE not provided via environment")
	}

	operatorDeployment, err := monitoring.GetServerlessOperatorDeployment(cl, namespace)
	if err != nil {
		return err
	}

	// If we upgrade from an old version we need to remove the old Service Monitor
	// that is not managed by OLM. See SRVCOM-1237 for more.
	if err = monitoring.RemoveOldServiceMonitorResourcesIfExist(namespace, cl); err != nil {
		return err
	}

	if err = monitoring.SetupClusterMonitoringRequirements(cl, operatorDeployment, namespace, nil); err != nil {
		return fmt.Errorf("failed to setup monitoring resources: %w", err)
	}

	co := &configv1.ClusterOperator{}
	if err = cl.Get(context.Background(), client.ObjectKey{Namespace: "", Name: consoleutil.ConsoleClusterOperatorName}, co); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch clusteroperator console: %w", err)
		}
	}
	if consoleutil.IsClusterOperatorAvailable(co.Status) {
		consoleutil.SetConsoleToInstalledStatus()
		if err := health.InstallHealthDashboard(cl); err != nil {
			return fmt.Errorf("failed to setup the Knative Health Status Dashboard: %w", err)
		}
	}
	return nil
}
