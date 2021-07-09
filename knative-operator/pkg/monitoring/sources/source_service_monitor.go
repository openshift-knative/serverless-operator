package sources

import (
	"fmt"

	mfclient "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/kmeta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SourceLabel     = "eventing.knative.dev/source"
	SourceNameLabel = "eventing.knative.dev/sourceName"
	SourceRoleLabel = "sources.knative.dev/role"
)

func SetupSourceServiceMonitorResources(client client.Client, instance *appsv1.Deployment) error {
	smManifest, err := sourceServiceMonitorManifest(client, instance)
	if err != nil {
		return err
	}
	return smManifest.Apply()
}

func RemoveSourceServiceMonitorResources(client client.Client, instance *appsv1.Deployment) error {
	smManifest, err := sourceServiceMonitorManifest(client, instance)
	if err != nil {
		return err
	}
	return smManifest.Delete()
}

func sourceServiceMonitorManifest(client client.Client, instance *appsv1.Deployment) (*mf.Manifest, error) {
	labels := instance.Spec.Selector.MatchLabels
	clientOptions := mf.UseClient(mfclient.NewClient(client))
	// Create service monitor resources for source
	smManifest, err := createServiceMonitorManifest(labels, instance.Name, instance.Namespace, clientOptions)
	if err != nil {
		return nil, err
	}
	if *smManifest, err = smManifest.Transform(mf.InjectOwner(instance)); err != nil {
		return nil, fmt.Errorf("unable to transform source service monitor manifest: %w", err)
	}

	return smManifest, nil
}

func createServiceMonitorManifest(labels map[string]string, depName string, ns string, options mf.Option) (*mf.Manifest, error) {
	var svU = &unstructured.Unstructured{}
	var smU = &unstructured.Unstructured{}
	sms := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      depName,
			Namespace: ns,
			Labels:    kmeta.CopyMap(labels),
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http-metrics",
				Port:       9090,
				TargetPort: intstr.FromInt(9090),
				Protocol:   "TCP",
			}},
			Selector: kmeta.CopyMap(labels),
		}}
	sms.Labels["name"] = sms.Name
	if err := scheme.Scheme.Convert(&sms, svU, nil); err != nil {
		return nil, err
	}
	sm := monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      depName,
			Namespace: ns,
			Labels:    kmeta.CopyMap(labels),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{Port: "http-metrics"}},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{ns},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"name": depName},
			},
		}}
	sm.Labels["name"] = sm.Name
	if err := scheme.Scheme.Convert(&sm, smU, nil); err != nil {
		return nil, err
	}
	smManifest, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{*svU, *smU}), options)
	if err != nil {
		return nil, err
	}
	return &smManifest, nil
}
