package kourier

import (
	"context"

	mf "github.com/jcrossley3/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log  = common.Log.WithName("kourier")
	path = "deploy/resources/kourier/kourier-latest.yaml"
)

// Apply applies Kourier resources.
func Apply(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Installing Kourier Ingress")
	manifest, err := mf.NewManifest(path, false, api)
	if err != nil {
		return err
	}
	transforms := []mf.Transformer{mf.InjectNamespace(ingressNamespace(instance.GetNamespace()))}

	if err := manifest.Transform(transforms...); err != nil {
		return err
	}
	return manifest.ApplyAll()
}

// Delete deletes Kourier resources.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Deleting Kourier Ingress")
	manifest, err := mf.NewManifest(path, false, api)
	if err != nil {
		return err
	}
	transforms := []mf.Transformer{mf.InjectNamespace(ingressNamespace(instance.GetNamespace()))}

	if err := manifest.Transform(transforms...); err != nil {
		return err
	}
	if err := manifest.DeleteAll(); err != nil {
		return err
	}

	log.Info("Deleting ingress namespace")
	ns := &v1.Namespace{}
	err = api.Get(context.TODO(), client.ObjectKey{Name: ingressNamespace(instance.GetNamespace())}, ns)
	if apierrors.IsNotFound(err) {
		// We can safely ignore this. There is nothing to do for us.
		return nil
	} else if err != nil {
		return err
	}
	return api.Delete(context.TODO(), ns)
}

func ingressNamespace(servingNamespace string) string {
	return servingNamespace + "-ingress"
}
