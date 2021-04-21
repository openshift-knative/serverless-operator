package monitoring

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestLoadPlatformEventingMonitoringManifests(t *testing.T) {
	manifests, err := GetEventingMonitoringPlatformManifests(&v1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{Namespace: eventingNamespace},
	})
	if err != nil {
		t.Errorf("Unable to load eventing monitoring platform manifests: %w", err)
	}
	if len(manifests) != 1 {
		t.Errorf("Got %d, want %d", len(manifests), 1)
	}
	resources := manifests[0].Resources()
	if len(resources) != 28 {
		t.Errorf("Got %d, want %d", len(resources), 28)
	}
	for _, u := range resources {
		kind := strings.ToLower(u.GetKind())
		switch kind {
		case "servicemonitor":
			if !eventingDeployments.Has(strings.TrimSuffix(u.GetName(), "-sm")) {
				t.Errorf("Service monitor with name %q not found", u.GetName())
			}
		case "service":
			if !eventingDeployments.Has(strings.TrimSuffix(u.GetName(), "-sm-service")) {
				t.Errorf("Service with name %q not found", u.GetName())
			}
		case "clusterrolebinding":
			if u.GetName() == "rbac-proxy-metrics-prom-rb" || u.GetName() == "rbac-proxy-reviews-prom-rb" {
				continue
			}
			if !eventingDeployments.Has(strings.TrimPrefix(u.GetName(), "rbac-proxy-reviews-prom-rb-")) {
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
