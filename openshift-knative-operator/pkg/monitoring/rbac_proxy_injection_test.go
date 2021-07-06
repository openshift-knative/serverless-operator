package monitoring

import (
	"fmt"
	"os"
	"testing"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestInjectRbacProxyContainerToDeployments(t *testing.T) {
	manifest, err := mf.NewManifest("testdata/serving-core-deployment.yaml")
	rbacImage := "registry.ci.openshift.org/origin/4.7:kube-rbac-proxy"
	os.Setenv(rbacProxyImageEnvVar, rbacImage)
	if err != nil {
		t.Errorf("Unable to load test manifest: %w", err)
	}
	transforms := []mf.Transformer{InjectRbacProxyContainerToDeployments(servingDeployments)}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		t.Errorf("Unable to transform test manifest: %w", err)
	}
	if len(manifest.Resources()) != 1 {
		t.Errorf("Got %d, want %d", len(manifest.Resources()), 1)
	}
	deployment := &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(&manifest.Resources()[0], deployment, nil); err != nil {
		t.Errorf("Unable to convert to deployment %w", err)
	}
	// Make sure we respect existing volumes (eg. controller gets extra volumes due to custom certs)
	if len(deployment.Spec.Template.Spec.Volumes) != 2 {
		t.Errorf("Got %d, want %d", len(deployment.Spec.Template.Spec.Volumes), 2)
	}
	cContainer := deployment.Spec.Template.Spec.Containers[0]
	if !envToString(cContainer.Env).Has("METRICS_PROMETHEUS_HOST:127.0.0.1") {
		t.Error("Component container does not set up the prometheus host to localhost")
	}
	rbacContainer := deployment.Spec.Template.Spec.Containers[1]
	if rbacContainer.Name != rbacContainerName {
		t.Errorf("Got %q, want %q", rbacContainer.Name, rbacContainerName)
	}
	if rbacContainer.Image != rbacImage {
		t.Errorf("Got %q, want %q", rbacContainer.Image, rbacImage)
	}
	// Make sure we define requests otherwise K8s hpa will complain
	if len(rbacContainer.Resources.Requests) != 2 {
		t.Errorf("Got %q, want %q", len(rbacContainer.Resources.Requests), 2)
	}
}
func envToString(vars []v1.EnvVar) sets.String {
	sVars := sets.String{}
	for _, v := range vars {
		sVars.Insert(fmt.Sprintf("%s:%s", v.Name, v.Value))
	}
	return sVars
}
