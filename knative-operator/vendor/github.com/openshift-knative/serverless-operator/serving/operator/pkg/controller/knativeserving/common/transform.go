package common

import (
	"os"

	mf "github.com/jcrossley3/manifestival"
	servingv1alpha1 "github.com/openshift-knative/serverless-operator/serving/operator/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

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
					if container.Name == u.GetName() && container.Image != image {
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

func configFromResource(instance *servingv1alpha1.KnativeServing) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "ConfigMap" {
			if data, ok := instance.Spec.Config[u.GetName()[len(`config-`):]]; ok {
				UpdateConfigMap(u, data, log)
			}
		}
		return nil
	}
}

func replaceQueueImage(u *unstructured.Unstructured) error {
	image := os.Getenv("IMAGE_QUEUE")
	if len(image) > 0 {
		switch u.GetKind() {
		case "Image":
			if u.GetName() == "queue-proxy" {
				x, _, _ := unstructured.NestedFieldNoCopy(u.Object, "spec", "image")
				if x != image {
					log.Info("Replacing queue-proxy", "image", image, "previous", x)
					if err := unstructured.SetNestedField(u.Object, image, "spec", "image"); err != nil {
						return err
					}
				}
			}
		case "ConfigMap":
			if u.GetName() == "config-deployment" {
				data := map[string]string{"queueSidecarImage": image}
				UpdateConfigMap(u, data, log)
			}
		}
	}
	return nil
}
