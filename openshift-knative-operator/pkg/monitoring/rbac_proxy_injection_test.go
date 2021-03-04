package monitoring

import (
	"fmt"
	"strings"
	"testing"

	mf "github.com/manifestival/manifestival"
	"github.com/manifestival/manifestival/fake"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestInjectRbacProxyContainerToDeployments(t *testing.T) {
	client := fake.New()
	manifest, err := mf.NewManifest("../testdata/serving-core-deployment.yaml", mf.UseClient(client))
	if err != nil {
		t.Errorf("Unable to load test manifest: %w", err)
	}
	transforms := []mf.Transformer{InjectRbacProxyContainerToDeployments()}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		t.Errorf("Unable to transform test manifest: %w", err)
	}
	if err := manifest.Apply(); err != nil {
		t.Errorf("Unable to apply the test manifest %w", err)
	}
	u := createDeployment("activator", servingNamespace)
	depU, err := client.Get(u)
	if err != nil {
		t.Errorf("Unable to get the deployment %w", err)
	}
	deployment := &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(depU, deployment, nil); err != nil {
		t.Errorf("Unable to convert deployment %w", err)
	}
	// Make sure we respect existing volumes (eg. controller gets extra volumes due to custom certs)
	if len(deployment.Spec.Template.Spec.Volumes) != 2 {
		t.Errorf("Got %d, want %d", len(deployment.Spec.Template.Spec.Volumes), 2)
	}
	cContainer := deployment.Spec.Template.Spec.Containers[0]
	if !strings.Contains(envToString(cContainer.Env), `"METRICS_PROMETHEUS_HOST":"127.0.0.1"`) {
		t.Error("Component container does not set up the prometheus host to localhost")
	}
	rbacContainer := deployment.Spec.Template.Spec.Containers[1]
	if rbacContainer.Name != rbacContainerName {
		t.Errorf("Got %q, want %q", rbacContainer.Name, rbacContainerName)
	}
	if rbacContainer.Image != fallbackImage {
		t.Errorf("Got %q, want %q", rbacContainer.Image, fallbackImage)
	}
	// Make sure we define requests otherwise K8s hpa will complain
	if len(rbacContainer.Resources.Requests) != 2 {
		t.Errorf("Got %q, want %q", len(rbacContainer.Resources.Requests), 2)
	}
}
func envToString(vars []v1.EnvVar) string {
	builder := strings.Builder{}
	for _, v := range vars {
		builder.WriteString(fmt.Sprintf("%q:%q,", v.Name, v.Value))
	}
	return builder.String()
}
func createDeployment(name string, ns string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind("Deployment")
	u.SetAPIVersion("apps/v1")
	u.SetName(name)
	u.SetNamespace(ns)
	return u
}
