package kourier

import (
	mf "github.com/jcrossley3/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log       = common.Log.WithName("kourier")
	manifests = []string{"deploy/resources/kourier/kourier-latest.yaml", "deploy/resources/kourier/kourier-openshift.yaml"}
)

// Apply applies Kourier resources.
func Apply(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Installing Kourier Ingress")
	for _, path := range manifests {
		if err := apply(instance, api, path); err != nil {
			log.Error(err, "Failed to apply %s: %v", path, err)
			return err
		}
	}
	return nil
}

func apply(instance *servingv1alpha1.KnativeServing, api client.Client, path string) error {
	manifest, err := mf.NewManifest(path, false, api)
	if err != nil {
		return err
	}
	// TODO: Use ingressNamespace(instance.Namespace)
	transforms = append(transforms, mf.InjectNamespace("knative-serving-ingress"))

	if err := manifest.Transform(transforms...); err != nil {
		return err
	}
	return manifest.ApplyAll()
}

// Delete deletes Kourier resources.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	for _, path := range manifests {
		manifest, err := mf.NewManifest(path, false, api)
		if err != nil {
			return err
		}
		if err := manifest.DeleteAll(); err != nil {
			return err
		}
	}
	return nil
}
