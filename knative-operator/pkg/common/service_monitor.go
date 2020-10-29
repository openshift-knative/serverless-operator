package common

import (
	"context"
	"errors"
	"fmt"
	"os"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	mfclient "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	kmeta "knative.dev/pkg/kmeta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EventingBrokerServiceMonitorPath     = "deploy/resources/broker-service-monitors.yaml"
	EventingSourceServiceMonitorPath     = "deploy/resources/source-service-monitor.yaml"
	EventingSourcePath                   = "deploy/resources/source-service.yaml"
	SourceLabel                          = "eventing.knative.dev/source"
	SourceNameLabel                      = "eventing.knative.dev/sourceName"
	SourceRoleLabel                      = "sources.knative.dev/role"
	TestEventingBrokerServiceMonitorPath = "TEST_EVENTING_BROKER_SERVICE_MONITOR_PATH"
	TestMonitor                          = "TEST_MONITOR"
	TestSourceServiceMonitorPath         = "TEST_SOURCE_SERVICE_MONITOR_PATH"
	TestSourceServicePath                = "TEST_SOURCE_SERVICE_PATH"
)

func SetupServerlessOperatorServiceMonitor(cfg *rest.Config, api client.Client, metricsPort int32, metricsHost string, operatorMetricsPort int32) error {
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
	service, err := metrics.CreateMetricsService(context.Background(), cfg, servicePorts)
	if err != nil {
		return fmt.Errorf("failed to create metrics service: %w", err)
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	metricsNamespace := os.Getenv(NamespaceEnvKey)
	if metricsNamespace == "" {
		return errors.New("NAMESPACE not provided via environment")
	}

	if _, err := metrics.CreateServiceMonitors(cfg, metricsNamespace, services); err != nil {
		if err == metrics.ErrServiceMonitorNotPresent {
			// If this operator is deployed to a cluster without the prometheus-operator running, it will return
			// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects")
			return nil
		}
		if apierrs.IsAlreadyExists(err) {
			// If the servicemonitor already exists, we don't want to report an error.
			return nil
		}
		return fmt.Errorf("failed to create service monitors: %w", err)
	}
	return nil
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config, metricsHost string, operatorMetricsPort int32) error {

	// If we dont use a custom list here, the typical call to get a filtered list of gvks using k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	// will end up with resources the Operator does not have access to eg. `Kind=LimitRange`
	// The list of resources returned is unrelated to our purposes here, thus the customization.
	gvkFilterList := []schema.GroupVersionKind{
		{
			Group:   "operator.knative.dev",
			Version: "v1alpha1",
			Kind:    "KnativeServing",
		},
		{
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

func SetupEventingBrokerServiceMonitors(client client.Client, instance *eventingv1alpha1.KnativeEventing) error {
	manifest, err := mf.NewManifest(getMonitorPath(TestEventingBrokerServiceMonitorPath, EventingBrokerServiceMonitorPath), mf.UseClient(mfclient.NewClient(client)))
	if err != nil {
		return fmt.Errorf("unable to parse broker service monitors: %w", err)
	}
	transforms := []mf.Transformer{mf.InjectOwner(instance), mf.InjectNamespace(instance.Namespace)}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		return fmt.Errorf("unable to transform broker service monitors manifest: %w", err)
	}
	if err := manifest.Apply(); err != nil {
		return err
	}
	return nil
}

func SetupSourceServiceMonitor(client client.Client, instance *appsv1.Deployment) error {
	labels := instance.Spec.Selector.MatchLabels

	clientOptions := mf.UseClient(mfclient.NewClient(client))
	// create service for the deployment
	manifest, err := mf.NewManifest(getMonitorPath(TestSourceServicePath, EventingSourcePath), clientOptions)
	if err != nil {
		return fmt.Errorf("unable to parse source service manifest: %w", err)
	}
	transforms := []mf.Transformer{updateService(labels, instance.Name), mf.InjectOwner(instance), mf.InjectNamespace(instance.Namespace)}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		return fmt.Errorf("unable to transform source service manifest: %w", err)
	}
	if err := manifest.Apply(); err != nil {
		return err
	}

	// get service back, needed for the UID and setting owner refs
	srv := &v1.Service{}
	if err := client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, srv); err != nil {
		return err
	}
	// create service monitor for source
	manifest, err = mf.NewManifest(getMonitorPath(TestSourceServiceMonitorPath, EventingSourceServiceMonitorPath), clientOptions)
	if err != nil {
		return fmt.Errorf("unable to parse source service monitor manifest: %w", err)
	}
	transforms = []mf.Transformer{updateServiceMonitor(labels, instance.Name), mf.InjectOwner(srv), mf.InjectNamespace(instance.Namespace)}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		return fmt.Errorf("unable to transform source service monitor manifest: %w", err)
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

func updateService(labels map[string]string, depName string) mf.Transformer {
	return func(resource *unstructured.Unstructured) error {
		if resource.GetKind() != "Service" {
			return nil
		}
		var svc = &v1.Service{}
		if err := scheme.Scheme.Convert(resource, svc, nil); err != nil {
			return err
		}
		svc.Name = depName
		svc.Labels = kmeta.CopyMap(labels)
		svc.Spec.Selector = kmeta.CopyMap(labels)
		svc.Labels["name"] = svc.Name
		return scheme.Scheme.Convert(svc, resource, nil)
	}
}

func updateServiceMonitor(labels map[string]string, depName string) mf.Transformer {
	return func(resource *unstructured.Unstructured) error {
		if resource.GetKind() != "ServiceMonitor" {
			return nil
		}
		var sm = &monitoringv1.ServiceMonitor{}
		if err := scheme.Scheme.Convert(resource, sm, nil); err != nil {
			return err
		}
		sm.Name = depName
		sm.Labels = kmeta.CopyMap(labels)
		sm.Spec.Selector = metav1.LabelSelector{
			MatchLabels: map[string]string{"name": sm.Name},
		}
		sm.Labels["name"] = sm.Name
		return scheme.Scheme.Convert(sm, resource, nil)
	}
}
