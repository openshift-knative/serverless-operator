package monitoring

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestLoadPlatformServingMonitoringManifests(t *testing.T) {
	manifests, err := GetServingMonitoringPlatformManifests(&v1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Namespace: servingNamespace},
	})
	if err != nil {
		t.Errorf("Unable to load serving monitoring platform manifests: %w", err)
	}
	if len(manifests) != 1 {
		t.Errorf("Got %d, want %d", len(manifests), 1)
	}
	resources := manifests[0].Resources()
	if len(resources) != 20 {
		t.Errorf("Got %d, want %d", len(resources), 20)
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
