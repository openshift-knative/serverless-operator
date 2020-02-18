package kourier

import (
	"context"
	"fmt"

	mf "github.com/jcrossley3/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
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
	instance.Status.MarkDependencyInstalling("Kourier")
	if err := api.Status().Update(context.TODO(), instance); err != nil {
		return err
	}

	log.Info("Installing Kourier Ingress")
	manifest, err := mf.NewManifest(path, false, api)
	if err != nil {
		return err
	}
	transforms := []mf.Transformer{mf.InjectNamespace(common.IngressNamespace(instance.GetNamespace()))}
	if err := manifest.Transform(transforms...); err != nil {
		return err
	}
	if err := manifest.ApplyAll(); err != nil {
		return err
	}
	if err := checkDeployments(&manifest, instance, api); err != nil {
		return err
	}
	log.Info("Kourier is ready")

	instance.Status.MarkDependenciesInstalled()
	return api.Status().Update(context.TODO(), instance)
}

// Check for deployments in knative-serving-ingress
// This function is copied from knativeserving_controller.go in serving-operator
func checkDeployments(manifest *mf.Manifest, instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Checking deployments")
	for _, u := range manifest.Resources {
		if u.GetKind() == "Deployment" {
			deployment := &appsv1.Deployment{}
			err := api.Get(context.TODO(), client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}, deployment)
			if err != nil {
				return err
			}
			for _, c := range deployment.Status.Conditions {
				if c.Type == appsv1.DeploymentAvailable && c.Status != v1.ConditionTrue {
					return fmt.Errorf("Deployment %q/%q not ready", u.GetName(), u.GetNamespace())
				}
			}
		}
	}
	return nil
}

// Delete deletes Kourier resources.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Deleting Kourier Ingress")
	manifest, err := mf.NewManifest(path, false, api)
	if err != nil {
		return err
	}
	transforms := []mf.Transformer{mf.InjectNamespace(common.IngressNamespace(instance.GetNamespace()))}

	if err := manifest.Transform(transforms...); err != nil {
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
