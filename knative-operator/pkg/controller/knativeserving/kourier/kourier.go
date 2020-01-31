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
	const path = "deploy/resources/kourier/kourier.yaml"

	manifest, err := mf.NewManifest(path, false, api)
	if err != nil {
		log.Error(err, "Unable to create Kourier Ingress install manifest")
		return err
	}
	transforms := []mf.Transformer{mf.InjectOwner(instance)}
	// let's hardcode this for now.
	transforms = append(transforms, mf.InjectNamespace("knative-serving-ingress"))

	if err := manifest.Transform(transforms...); err != nil {
		log.Error(err, "Unable to transform Kourier Ingress manifest")
		return err
	}
	if err := manifest.ApplyAll(); err != nil {
		log.Error(err, "Unable to install Kourier Ingress")
		return err
	}
	return nil
}
