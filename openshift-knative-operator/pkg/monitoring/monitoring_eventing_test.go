package monitoring

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func testLoadPlatformEventingMonitoringManifests(t *testing.T, supportJobSink bool) {
	comp := &operatorv1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{Namespace: eventingNamespace},
	}
	manifests, err := GetEventingMonitoringPlatformManifests(comp)
	if err != nil {
		t.Errorf("Unable to load eventing monitoring platform manifests: %v", err)
	}
	if len(manifests) != 1 {
		t.Errorf("Got %d, want %d", len(manifests), 1)
	}
	resources := manifests[0].Resources()

	deployments := eventingDeployments
	if !supportJobSink {
		deployments = eventingDeployments.Clone()
		deployments.Delete("job-sink")
	}

	// We create a service monitor and a service monitor service per deployment: len(eventingDeployments)*2 resources.
	// One clusterrolebinding (except for mt-broker-controller) per deployment for allowing tokenreviews, subjectaccessreviews
	// to be used by kube proxy. All but one deployments have a different sa: len(eventingDeployments) -1 resources.
	// RBAC resources from rbac-proxy.yaml: 5 resources that don't depend on the deployments number.
	expectedEventingMonitoringResources := len(deployments)*2 + len(deployments) - 1 + 5

	if len(resources) != expectedEventingMonitoringResources {
		t.Errorf("Got %d, want %d", len(resources), expectedEventingMonitoringResources)
	}
	for _, u := range resources {
		kind := strings.ToLower(u.GetKind())
		switch kind {
		case "servicemonitor":
			if !deployments.Has(strings.TrimSuffix(u.GetName(), "-sm")) {
				t.Errorf("Service monitor with name %q not found", u.GetName())
			}
		case "service":
			if !deployments.Has(strings.TrimSuffix(u.GetName(), "-sm-service")) {
				t.Errorf("Service with name %q not found", u.GetName())
			}
		case "clusterrolebinding":
			if u.GetName() == "rbac-proxy-metrics-prom-rb" || u.GetName() == "rbac-proxy-reviews-prom-rb" {
				continue
			}
			if !deployments.Has(strings.TrimPrefix(u.GetName(), "rbac-proxy-reviews-prom-rb-")) {
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

func TestLoadPlatformEventingMonitoringManifests(t *testing.T) {
	t.Setenv("KO_DATA_PATH", "testdata/jobsink")
	testLoadPlatformEventingMonitoringManifests(t, true)
}

func TestLoadPlatformEventingMonitoringManifestsJobSink(t *testing.T) {
	t.Setenv("KO_DATA_PATH", "testdata/nojobsink")
	testLoadPlatformEventingMonitoringManifests(t, false)
}
