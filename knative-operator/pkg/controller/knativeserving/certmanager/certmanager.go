package certmanager

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
	log    = common.Log.WithName("cert-manager")
	mfpath = "deploy/resources/cert-manager/cert-manager.yaml"

	defaultIssuer = "deploy/resources/cert-manager/clusterissuer.yaml"
	defaultSecret = "deploy/resources/cert-manager/secret.yaml"
)

// Apply applies Kourier resources.
func Apply(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	manifest, err := mf.NewManifest(mfpath, false, api)
	if err != nil {
		return err
	}
	log.Info("Installing cert manager")
	if err := manifest.ApplyAll(); err != nil {
		return err
	}

	manifest, err = mf.NewManifest(defaultSecret, false, api)
	if err != nil {
		return err
	}
	log.Info("Installing CA for issuer")
	if err := manifest.ApplyAll(); err != nil {
		return err
	}

	manifest, err = mf.NewManifest(defaultIssuer, false, api)
	if err != nil {
		return err
	}
	log.Info("Installing cluster Issuer")
	if err := manifest.ApplyAll(); err != nil {
		return err
	}

	return nil
}

// Delete deletes Kourier resources.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Deleting default issuer")
	manifest, err := mf.NewManifest(defaultIssuer, false, api)
	if err != nil {
		return err
	}
	if err := manifest.DeleteAll(); err != nil {
		return err
	}

	log.Info("Deleting Cert Manager")
	manifest, err = mf.NewManifest(defaultSecret, false, api)
	if err != nil {
		return err
	}
	if err := manifest.DeleteAll(); err != nil {
		return err
	}

	log.Info("Deleting Cert Manager")
	manifest, err = mf.NewManifest(mfpath, false, api)
	if err != nil {
		return err
	}
	if err := manifest.DeleteAll(); err != nil {
		return err
	}

	log.Info("Deleting cert manager namespace")
	ns := &v1.Namespace{}
	err = api.Get(context.TODO(), client.ObjectKey{Name: "cert-manager"}, ns)
	if apierrors.IsNotFound(err) {
		// We can safely ignore this. There is nothing to do for us.
		return nil
	} else if err != nil {
		return err
	}
	return api.Delete(context.TODO(), ns)
}
