package kourier

import (
	"context"
	"fmt"
	"os"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = common.Log.WithName("kourier")

// Apply applies Kourier resources.
func Apply(instance *servingv1alpha1.KnativeServing, api client.Client, scheme *runtime.Scheme) error {
	manifest, err := manifest(common.IngressNamespace(instance.GetNamespace()), api, instance, scheme)
	if err != nil {
		return fmt.Errorf("failed to load kourier manifest: %w", err)
	}
	log.Info("Installing Kourier Ingress")
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply kourier manifest: %w", err)
	}
	if err := checkDeployments(&manifest, instance, api); err != nil {
		instance.Status.MarkDependencyInstalling("Kourier")
		return fmt.Errorf("failed to check deployments: %w", err)
	}
	log.Info("Kourier is ready")
	instance.Status.MarkDependenciesInstalled()
	return nil
}

// Check for deployments in knative-serving-ingress
// This function is copied from knativeserving_controller.go in serving-operator
func checkDeployments(manifest *mf.Manifest, instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Checking deployments")
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
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
	return nil
}

// Delete deletes Kourier resources.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client, scheme *runtime.Scheme) error {
	log.Info("Deleting Kourier Ingress")
	manifest, err := manifest(common.IngressNamespace(instance.GetNamespace()), api, instance, scheme)
	if err != nil {
		return fmt.Errorf("failed to load kourier manifest: %w", err)
	}

	if err := manifest.Delete(); err != nil {
		return fmt.Errorf("failed to delete kourier manifest: %w", err)
	}

	log.Info("Deleting ingress namespace")
	ns := &v1.Namespace{}
	err = api.Get(context.TODO(), client.ObjectKey{Name: common.IngressNamespace(instance.GetNamespace())}, ns)
	if apierrors.IsNotFound(err) {
		// We can safely ignore this. There is nothing to do for us.
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to fetch ingress namespace: %w", err)
	}
	if err := api.Delete(context.TODO(), ns); err != nil {
		return fmt.Errorf("failed to remove ingress namespace: %w", err)
	}
	return nil
}

// replaceImageFromEnvironment replaces Kourier images with the images specified by env value.
// This func is copied from serving/operator/pkg/controller/knativeserving/common/transform.go and modified.
func replaceImageFromEnvironment(prefix string, scheme *runtime.Scheme) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" {
			image := os.Getenv(prefix + u.GetName())
			if len(image) > 0 {
				deploy := &appsv1.Deployment{}
				if err := scheme.Convert(u, deploy, nil); err != nil {
					return err
				}
				containers := deploy.Spec.Template.Spec.Containers
				for i, container := range containers {
					if "3scale-"+container.Name == u.GetName() && container.Image != image {
						log.Info("Replacing", "deployment", container.Name, "image", image, "previous", container.Image)
						containers[i].Image = image
						break
					}
				}
				if err := scheme.Convert(deploy, u, nil); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func replaceDeploymentInstanceCount(availability *servingv1alpha1.HighAvailability,
	scheme *runtime.Scheme) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" {
			if availability != nil && availability.Replicas > 1 {
				deploy := &appsv1.Deployment{}
				if err := scheme.Convert(u, deploy, nil); err != nil {
					return err
				}
				deploy.Spec.Replicas = &availability.Replicas
				if err := scheme.Convert(deploy, u, nil); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// RawManifest returns kourier raw manifest without transformations
func RawManifest(apiclient client.Client) (mf.Manifest, error) {
	return mfc.NewManifest(manifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
}

// manifest returns kourier manifest after transformed
func manifest(namespace string, apiclient client.Client, instance *servingv1alpha1.KnativeServing, scheme *runtime.Scheme) (mf.Manifest, error) {
	manifest, err := RawManifest(apiclient)
	if err != nil {
		return mf.Manifest{}, err
	}
	transforms := []mf.Transformer{
		mf.InjectNamespace(namespace),
		replaceImageFromEnvironment("IMAGE_", scheme),
		func(u *unstructured.Unstructured) error {
			u.SetAnnotations(map[string]string{
				common.ServingOwnerName:      instance.Name,
				common.ServingOwnerNamespace: instance.Namespace,
			})
			return nil
		},
		replaceDeploymentInstanceCount(instance.Spec.HighAvailability, scheme),
	}
	return manifest.Transform(transforms...)
}

func manifestPath() string {
	return os.Getenv("KOURIER_MANIFEST_PATH")
}
