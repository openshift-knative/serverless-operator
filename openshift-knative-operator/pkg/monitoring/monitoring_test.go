package monitoring

import (
	"os"
	"strings"
	"testing"

	mf "github.com/manifestival/manifestival"
	"github.com/manifestival/manifestival/fake"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const (
	servingNamespace  = "knative-serving"
	eventingNamespace = "knative-eventing"
)

func init() {
	os.Setenv(smRbacManifestPath, "../testdata/rbac-proxy.yaml")
}

func TestSetupServingRbacTransformation(t *testing.T) {
	client := fake.New()
	manifest, err := mf.NewManifest("../testdata/rbac.yaml", mf.UseClient(client))
	if err != nil {
		t.Errorf("Unable to load test manifest: %w", err)
	}
	transforms := []mf.Transformer{InjectNamespaceWithSubject(servingNamespace, OpenshiftMonitoringNamespace)}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		t.Errorf("Unable to transform test manifest: %w", err)
	}
	if err := manifest.Apply(); err != nil {
		t.Errorf("Unable to apply the test manifest %w", err)
	}
	u := createRole(prometheusRoleName, servingNamespace)
	_, err = client.Get(u)
	if err != nil {
		t.Errorf("Unable to get the role %w", err)
	}
	u = createRole("test-role", "default")
	_, err = client.Get(u)
	if err != nil {
		t.Errorf("Unable to get the role %w", err)
	}
	u = createClusterRole()
	_, err = client.Get(u)
	if err != nil {
		t.Errorf("Unable to get the cluster role %w", err)
	}
	u = createRoleBinding(prometheusRoleName, servingNamespace)
	resultRoleBinding, err := client.Get(u)
	if err != nil {
		t.Errorf("Unable to get the rolebinding %w", err)
	}
	checkSubjects(t, resultRoleBinding.Object, OpenshiftMonitoringNamespace)
	u = createRoleBinding("test-rb", "default")
	resultRoleBinding, err = client.Get(u)
	if err != nil {
		t.Errorf("Unable to get the rolebinding %w", err)
	}
	checkSubjects(t, resultRoleBinding.Object, "default")
	u = createClusterRoleBinding()
	resultClusterRoleBinding, err := client.Get(u)
	if err != nil {
		t.Errorf("Unable to get the cluster rolebinding %w", err)
	}
	checkSubjects(t, resultClusterRoleBinding.Object, OpenshiftMonitoringNamespace)
	// Make sure unrelated resources are not touched
	u = createService("activator-sm-service", "test")
	_, err = client.Get(u)
	if err != nil {
		t.Errorf("Unable to get the service %w", err)
	}
}

func TestLoadPlatformServingMonitoringManifests(t *testing.T) {
	manifests, err := GetCompMonitoringPlatformManifests(&v1alpha1.KnativeServing{ObjectMeta: v1.ObjectMeta{Namespace: servingNamespace}})
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
			if !servingComponents.Has(strings.TrimSuffix(u.GetName(), "-sm")) {
				t.Errorf("Service monitor with name %q not found", u.GetName())
			}
		case "service":
			if !servingComponents.Has(strings.TrimSuffix(u.GetName(), "-sm-service")) {
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

func TestLoadPlatformEventingMonitoringManifests(t *testing.T) {
	manifests, err := GetCompMonitoringPlatformManifests(&v1alpha1.KnativeEventing{ObjectMeta: v1.ObjectMeta{Namespace: eventingNamespace}})
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
			if !eventingComponents.Has(strings.TrimSuffix(u.GetName(), "-sm")) {
				t.Errorf("Service monitor with name %q not found", u.GetName())
			}
		case "service":
			if !eventingComponents.Has(strings.TrimSuffix(u.GetName(), "-sm-service")) {
				t.Errorf("Service with name %q not found", u.GetName())
			}
		case "clusterrolebinding":
			if u.GetName() == "rbac-proxy-metrics-prom-rb" || u.GetName() == "rbac-proxy-reviews-prom-rb" {
				continue
			}
			if !eventingComponents.Has(strings.TrimPrefix(u.GetName(), "rbac-proxy-reviews-prom-rb-")) {
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

func checkSubjects(t *testing.T, object map[string]interface{}, ns string) {
	subjects, _, _ := unstructured.NestedFieldNoCopy(object, "subjects")
	subjs := subjects.([]interface{})
	m := subjs[0].(map[string]interface{})
	if m["namespace"] != ns {
		t.Errorf("Got %q, want %q", m["namespace"], ns)
	}
}

func createService(name string, ns string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind("Service")
	u.SetAPIVersion("v1")
	u.SetName(name)
	u.SetNamespace(ns)
	return u
}

func createRole(name string, ns string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind("Role")
	u.SetAPIVersion("rbac.authorization.k8s.io/v1")
	u.SetName(name)
	u.SetNamespace(ns)
	return u
}

func createClusterRole() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind("ClusterRole")
	u.SetAPIVersion("rbac.authorization.k8s.io/v1")
	u.SetName(prometheusClusterRoleName)
	return u
}

func createRoleBinding(name string, ns string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind("RoleBinding")
	u.SetAPIVersion("rbac.authorization.k8s.io/v1")
	u.SetName(name)
	u.SetNamespace(ns)
	return u
}

func createClusterRoleBinding() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind("ClusterRoleBinding")
	u.SetAPIVersion("rbac.authorization.k8s.io/v1")
	u.SetName(prometheusClusterRoleName + "-rb")
	return u
}
