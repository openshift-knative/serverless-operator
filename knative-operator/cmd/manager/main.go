package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
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

	configv1 "github.com/openshift/api/config/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapr "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/consoleutil"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
	healthPort  int32 = 8687
	pprofHost         = "127.0.0.1"
	pprofPort   int32 = 8008
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

	// Setup all Webhooks
	disableHTTP2 := func(c *tls.Config) { c.NextProtos = []string{"http/1.1"} }
	hookServer := webhook.NewServer(webhook.Options{
		Port:       9876,
		CertDir:    "/apiserver.local.config/certificates",
		CertName:   "apiserver.crt",
		KeyName:    "apiserver.key",
		TLSOpts:    []func(config *tls.Config){disableHTTP2},
		WebhookMux: nil,
	})

	// If empty the pprof serving is disabled
	pprofAddress := ""
	enable := false

	if enable, err = strconv.ParseBool(os.Getenv("ENABLE_PPROF")); err != nil {
		log.Error(err, "unable to parse ENABLE_PPROF")
		os.Exit(1)
	} else if enable {
		pprofAddress = fmt.Sprintf("%s:%d", pprofHost, pprofPort)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		LeaderElection:   true,
		LeaderElectionID: "knative-serving-openshift-lock",
		Metrics: metricsserver.Options{
			BindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		},
		HealthProbeBindAddress: fmt.Sprintf(":%d", healthPort),
		WebhookServer:          hookServer,
		PprofBindAddress:       pprofAddress,
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

	decoder := admission.NewDecoder(mgr.GetScheme())

	// This call adds the server to the manager as a Runnable
	_ = mgr.GetWebhookServer()
	// Serving Webhooks
	hookServer.Register("/mutate-knativeservings", &webhook.Admission{Handler: knativeserving.NewConfigurator(decoder)})
	hookServer.Register("/validate-knativeservings", &webhook.Admission{Handler: knativeserving.NewValidator(mgr.GetClient(), decoder)})
	// Eventing Webhooks
	hookServer.Register("/mutate-knativeeventings", &webhook.Admission{Handler: knativeeventing.NewConfigurator(decoder)})
	hookServer.Register("/validate-knativeeventings", &webhook.Admission{Handler: knativeeventing.NewValidator(mgr.GetClient(), decoder)})
	// Kafka Webhooks
	hookServer.Register("/validate-knativekafkas", &webhook.Admission{Handler: knativekafka.NewValidator(mgr.GetClient(), decoder)})

	if err := setupServerlessOperatorMonitoring(cfg); err != nil {
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

func setupServerlessOperatorMonitoring(cfg *rest.Config) error {
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
	// If we upgrade from an old version we need to remove the old Service Monitor
	// that is not managed by OLM. It can be removed after SRVKE-1510 is out.
	if err = monitoring.RemoveOldPingSourceServiceMonitorResourcesIfExist(cl); err != nil {
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
