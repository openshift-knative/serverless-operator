package kourier

import (
	mf "github.com/jcrossley3/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = common.Log.WithName("kourier")

func ApplyKourier(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Installing Kourier Ingress")
	resources := []string{"deploy/resources/kourier/kourier.yaml", "deploy/resources/kourier/kourier_openshift.yaml"}
	for _, path := range resources {
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
	transforms := []mf.Transformer{mf.InjectOwner(instance)}
	// TODO: Use ingressNamespace(instance.Namespace)
	transforms = append(transforms, mf.InjectNamespace("knative-serving-ingress"))

	if err := manifest.Transform(transforms...); err != nil {
		return err
	}
	return manifest.ApplyAll()
}
