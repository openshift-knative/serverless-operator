package monitoring

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestLoadPlatformServingMonitoringManifests(t *testing.T) {
	manifests, err := GetServingMonitoringPlatformManifests(&operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Namespace: servingNamespace},
	})
	if err != nil {
		t.Errorf("Unable to load serving monitoring platform manifests: %v", err)
	}
	if len(manifests) != 1 {
		t.Errorf("Got %d, want %d", len(manifests), 1)
	}
	resources := manifests[0].Resources()

	// We create a service monitor and a service monitor service per deployment: len(servingDeployments)*2 resources.
	// One clusterrolebinding for allowing tokenreviews, subjectaccessreviews
	// to be used by kube proxy. All deployments share the same sa: 1 resource.
	// RBAC resources from rbac-proxy.yaml: 5 resources that don't depend on the deployments number.
	expectedServingMonitoringResources := len(servingDeployments)*2 + 5 + 1

	if len(resources) != expectedServingMonitoringResources {
		t.Errorf("Got %d, want %d", len(resources), expectedServingMonitoringResources)
	}
	for _, u := range resources {
		kind := strings.ToLower(u.GetKind())
		switch kind {
		case "servicemonitor":
			if !servingDeployments.Has(strings.TrimSuffix(u.GetName(), "-sm")) {
				t.Errorf("Service monitor with name %q not found", u.GetName())
			}
		case "service":
			if !servingDeployments.Has(strings.TrimSuffix(u.GetName(), "-sm-service")) {
				t.Errorf("Service with name %q not found", u.GetName())
			}
		case "clusterrolebinding":
			if u.GetName() == "rbac-proxy-metrics-prom-rb" || u.GetName() == "rbac-proxy-reviews-prom-rb" {
				continue
			}
			if strings.TrimPrefix(u.GetName(), "rbac-proxy-reviews-prom-rb-") != "controller" {
				t.Errorf("Clusterrolebinding with name %q not found", u.GetName())
			}
		case "role":
			if u.GetName() != "knative-prometheus-k8s" {
				t.Errorf("Uknown role %q", u.GetName())
			}
		case "rolebinding":
			if u.GetName() != "knative-prometheus-k8s" {
				t.Errorf("Uknown rolebinding %q", u.GetName())
			}
			checkSubjects(t, u.Object, OpenshiftMonitoringNamespace)
		}
	}
}
