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
	"knative.dev/pkg/kmeta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EventingSourceServiceMonitorPath = "deploy/resources/monitoring/source-service-monitor.yaml"
	EventingSourcePath               = "deploy/resources/monitoring/source-service.yaml"
	SourceLabel                      = "eventing.knative.dev/source"
	SourceNameLabel                  = "eventing.knative.dev/sourceName"
	SourceRoleLabel                  = "sources.knative.dev/role"
	TestMonitor                      = "TEST_MONITOR"
	TestSourceServiceMonitorPath     = "TEST_SOURCE_SERVICE_MONITOR_PATH"
	TestSourceServicePath            = "TEST_SOURCE_SERVICE_PATH"
)

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
	return manifest.Apply()
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
