package kourier

import (
	"os"
	"testing"

	mfc "github.com/manifestival/controller-runtime-client"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReplaceImageFromEnvironment(t *testing.T) {
	api := fake.NewFakeClient()
	scheme := scheme.Scheme

	wantControlImage := "foo/bar:control"
	wantGatewayImage := "foo/bar:gateway"
	wantControlEnv := "knative-serving-ingress"
	os.Setenv("IMAGE_SERVING_3scale-kourier-control", wantControlImage)
	os.Setenv("IMAGE_SERVING_3scale-kourier-gateway", wantGatewayImage)

	manifest, err := mfc.NewManifest("testdata/kourier-latest.yaml", api)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	manifest, err = manifest.Transform(replaceImageFromEnvironment("IMAGE_SERVING_", scheme))
	if err != nil {
		t.Fatalf("Failed to transform manifest: %v", err)
	}

	manifest, err = manifest.Transform(replaceEnvValue("knative-serving-ingress", scheme))
	if err != nil {
		t.Fatalf("Failed to transform manifest: %v", err)
	}

	for _, resource := range manifest.Resources() {

		if resource.GetKind() == "Deployment" {
			deploy := &appsv1.Deployment{}
			if err := scheme.Convert(&resource, deploy, nil); err != nil {
				t.Fatalf("Failed to convert resource to deployment: %v", err)
			}
			image := deploy.Spec.Template.Spec.Containers[0].Image
			envs := deploy.Spec.Template.Spec.Containers[0].Env
			env := ""
			for i := range envs {
				if envs[i].Name == "KOURIER_GATEWAY_NAMESPACE" {
					env = envs[i].Value
				}
			}

			if deploy.Name == "3scale-kourier-control" && image != wantControlImage {
				t.Errorf("Image = %s, want %s", image, wantControlImage)
			}
			if deploy.Name == "3scale-kourier-control" && env != wantControlEnv {
				t.Errorf("KOURIER_GATEWAY_NAMESPACE = %s, want %s", env, wantControlEnv)
			}

			if deploy.Name == "3scale-kourier-gateway" && image != wantGatewayImage {
				t.Errorf("Image = %s, want %s", image, wantGatewayImage)
			}
		}
	}
}
