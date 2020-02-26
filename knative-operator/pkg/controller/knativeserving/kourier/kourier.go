package kourier

import (
	"context"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = common.Log.WithName("kourier")

// Apply applies Kourier resources.
func Apply(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	manifest, err := common.KourierManifest(instance, api)
	if err != nil {
		return err
	}
	if instance.Status.IsFullySupported() {
		// TODO: verify deployed kourier is not different from kourier-latest.yaml accurately.
		if err := common.CheckDeployments(&manifest, instance, api); err == nil {
			return nil
		}
	}

	// Us reaching here means we need to do something and/or wait longer.
	instance.Status.MarkDependencyInstalling("Kourier")
	if err := api.Status().Update(context.TODO(), instance); err != nil {
		return err
	}

	log.Info("Installing Kourier Ingress")
	if err := manifest.ApplyAll(); err != nil {
		return err
	}
	if err := common.CheckDeployments(&manifest, instance, api); err != nil {
		return err
	}
	log.Info("Kourier is ready")

	instance.Status.MarkDependenciesInstalled()
	return api.Status().Update(context.TODO(), instance)
}

// Delete deletes Kourier resources.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Deleting Kourier Ingress")
	manifest, err := common.KourierManifest(instance, api)
	if err != nil {
		return err
	}
	if err := manifest.DeleteAll(); err != nil {
		return err
	}

	log.Info("Deleting ingress namespace")
	ns := &v1.Namespace{}
	err = api.Get(context.TODO(), client.ObjectKey{Name: common.IngressNamespace(instance.GetNamespace())}, ns)
	if apierrors.IsNotFound(err) {
		// We can safely ignore this. There is nothing to do for us.
		return nil
	} else if err != nil {
		return err
	}
	return api.Delete(context.TODO(), ns)
}
