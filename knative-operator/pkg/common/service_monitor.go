package common

import (
	"context"
	"fmt"
	"os"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	mfclient "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EventingBrokerServiceMonitorPath     = "deploy/resources/broker-service-monitors.yaml"
	EventingPingSourceMonitorPath        = "deploy/resources/ping_source_service_monitor.yaml"
	testEventingBrokerServiceMonitorPath = "TEST_EVENTING_BROKER_SERVICE_MONITOR_PATH"
	testPingSourceServiceMonitorPath     = "TEST_PING_SOURCE_SERVICE_MONITOR_PATH"
)

func SetupServerlessOperatorServiceMonitor(ctx context.Context, cfg *rest.Config, api client.Client, metricsPort int32, metricsHost string, operatorMetricsPort int32) error {
	// Commented below to avoid a stream of these errors at startup:
	// E1021 22:50:03.372487       1 reflector.go:134] github.com/operator-framework/operator-sdk/pkg/kube-metrics/collector.go:67: Failed to list *unstructured.Unstructured: the server could not find the requested resource
	if err := serveCRMetrics(cfg, metricsHost, operatorMetricsPort); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	metricsNamespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		log.Error(err, "failed to get metrics namespace")
		return err
	}
	_, err = metrics.CreateServiceMonitors(cfg, metricsNamespace, services)

	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
	return err
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config, metricsHost string, operatorMetricsPort int32) error {

	// If we dont use a custom list here, the typical call to get a filtered list of gvks using k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	// will end up with resources the Operator does not have access to eg. `Kind=LimitRange`
	// The list of resources returned is unrelated to our purposes here, thus the customization.
	gvkFilterList := []schema.GroupVersionKind{
		schema.GroupVersionKind{
			Group:   "operator.knative.dev",
			Version: "v1alpha1",
			Kind:    "KnativeServing",
		},
		schema.GroupVersionKind{
			Group:   "operator.knative.dev",
			Version: "v1alpha1",
			Kind:    "KnativeEventing",
		},
	}
	// To generate metrics in other namespaces, add the values below.
	// This is due to this bug: https://github.com/operator-framework/operator-sdk/issues/2494
	// For the workaround check here: https://github.com/operator-framework/operator-sdk/pull/2601/files#r396745465
	// and https://github.com/shipwright-io/build/pull/73
	// In order to avoid getting a bad value we avoid using k8sutil.GetWatchNamespace() that gets the value from the WATCH_NAMESPACE
	// env var. That value is by default "" but user may change it, affecting the metrics endpoint.
	namespaces := []string{""}
	// Generate and serve custom resource specific metrics.
	err := kubemetrics.GenerateAndServeCRMetrics(cfg, namespaces, gvkFilterList, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}

func SetupEventingServiceMonitors(client client.Client, namespace string, instance *eventingv1alpha1.KnativeEventing) error {
	manifest, err := mf.NewManifest(getMonitorPath(testEventingBrokerServiceMonitorPath, EventingBrokerServiceMonitorPath), mf.UseClient(mfclient.NewClient(client)))
	if err != nil {
		return fmt.Errorf("unable to parse broker service monitors: %w", err)
	}
	transforms := []mf.Transformer{mf.InjectOwner(instance), mf.InjectNamespace(namespace)}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		return fmt.Errorf("unable to transform broker service monitors manifest: %w", err)
	}
	// this is required because Apply will fail with not known resource later on
	monitor := &monitoringv1.ServiceMonitor{}
	var SchemeGroupVersion = schema.GroupVersion{Group: "monitoring.coreos.com", Version: "v1"}
	scheme.Scheme.AddKnownTypes(SchemeGroupVersion, monitor)
	if err := manifest.Apply(); err != nil {
		return err
	}
	manifest, err = mf.NewManifest(getMonitorPath(testPingSourceServiceMonitorPath, EventingPingSourceMonitorPath), mf.UseClient(mfclient.NewClient(client)))
	if err != nil {
		return fmt.Errorf("unable to parse ping source service monitor: %w", err)
	}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		return fmt.Errorf("unable to transform ping source service monitor manifest: %w", err)
	}
	if err := manifest.Apply(); err != nil {
		return err
	}
	return nil
}

func getMonitorPath(envVar string, defaultVal string) string {
	path := os.Getenv(envVar)
	if path == "" {
		return defaultVal
	}
	return path
}
