package common

import (
	"context"
	"fmt"
	"os"

	mfclient "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	v1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/kmeta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EventingBrokerServiceMonitorPath     = "deploy/resources/monitoring/broker-service-monitors.yaml"
	EventingSourceServiceMonitorPath     = "deploy/resources/monitoring/source-service-monitor.yaml"
	EventingSourcePath                   = "deploy/resources/monitoring/source-service.yaml"
	ServingRbacProxyRolesPath            = "deploy/resources/monitoring/role-proxy-roles.yaml"
	ServingRbacProxyRolesPathEnv         = "SERVING_RBAC_PROXY_ROLES_PATH"
	SourceLabel                          = "eventing.knative.dev/source"
	SourceNameLabel                      = "eventing.knative.dev/sourceName"
	SourceRoleLabel                      = "sources.knative.dev/role"
	TestEventingBrokerServiceMonitorPath = "TEST_EVENTING_BROKER_SERVICE_MONITOR_PATH"
	TestMonitor                          = "TEST_MONITOR"
	TestSourceServiceMonitorPath         = "TEST_SOURCE_SERVICE_MONITOR_PATH"
	TestSourceServicePath                = "TEST_SOURCE_SERVICE_PATH"
)

func SetupServerlessOperatorServiceMonitor(cfg *rest.Config, api client.Client, metricsPort int32) error {
	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}}}
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

func SetupEventingBrokerServiceMonitors(client client.Client, instance *v1alpha1.KnativeEventing) error {
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
	if err := client.Get(context.Background(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, srv); err != nil {
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

func SetupServingControlPlaneServiceMonitors(api client.Client, instance *v1alpha1.KnativeServing) error {
	serviceNames := []string{"activator", "autoscaler", "autoscaler-hpa", "controller", "domain-mapping", "domainmapping-webhook", "webhook"}
	for _, sName := range serviceNames {
		// Create service for service monitor
		serv := &v1.Service{}
		serv.Name = sName + "-sm-service"
		serv.Namespace = "knative-serving"
		serv.Labels = map[string]string{"name": sName + "-sm-service"}
		serv.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(instance, instance.GroupVersionKind())}
		serv.Spec = v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "https",
					Port:       int32(8444),
					TargetPort: intstr.FromInt(8444),
				}},
			Selector: map[string]string{"app": sName},
		}
		err := api.Create(context.Background(), serv)
		if err != nil && !apierrs.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create service monitor service: %w", err)
		}
		// Create service monitor
		var sm = &monitoringv1.ServiceMonitor{}
		sm.Name = sName + "-sm"
		sm.Namespace = "knative-serving"
		sm.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(instance, instance.GroupVersionKind())}
		sm.Spec = monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					Port:            "https",
					Scheme:          "https",
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					TLSConfig: &monitoringv1.TLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							InsecureSkipVerify: true,
						}},
				}},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{sm.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"name": sName + "-sm-service"},
			}}

		if err := api.Create(context.Background(), sm); err != nil {
			if err == metrics.ErrServiceMonitorNotPresent {
				// If this operator is deployed to a cluster without the prometheus-operator running, it will return
				// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
				log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects")
				return nil
			}
			if apierrs.IsAlreadyExists(err) {
				// If the servicemonitor already exists, we don't want to report an error.
				continue
			} else {
				return fmt.Errorf("failed to create service monitor: %w", err)
			}
		}
	}
	return nil
}

func SetupRbacProxyRequirements(client client.Client, instance *v1alpha1.KnativeServing) error {
	clientOptions := mf.UseClient(mfclient.NewClient(client))
	manifest, err := mf.NewManifest(getMonitorPath(ServingRbacProxyRolesPathEnv, ServingRbacProxyRolesPath), clientOptions)
	if err != nil {
		return fmt.Errorf("unable to parse rbax proxy manifest: %w", err)
	}
	if err := manifest.Apply(); err != nil {
		return err
	}
	return nil
}

func DeleteRbacProxyRequirements(client client.Client, instance *v1alpha1.KnativeServing) error {
	clientOptions := mf.UseClient(mfclient.NewClient(client))
	manifest, err := mf.NewManifest(getMonitorPath(ServingRbacProxyRolesPathEnv, ServingRbacProxyRolesPath), clientOptions)
	if err != nil {
		return fmt.Errorf("unable to parse rbax proxy manifest: %w", err)
	}
	if err := manifest.Delete(); err != nil {
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
