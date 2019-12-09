/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package common

import (
	"strings"

	mf "github.com/jcrossley3/manifestival"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	caching "knative.dev/caching/pkg/apis/caching/v1alpha1"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

func init() {
	caching.AddToScheme(scheme.Scheme)
}

var (
	// The string to be replaced by the container name
	containerNameVariable = "${NAME}"
)

func DeploymentTransform(instance *servingv1alpha1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Update the deployment with the new registry and tag
		if u.GetKind() == "Deployment" {
			return updateDeployment(instance, u, log)
		}
		return nil
	}
}

func ImageTransform(instance *servingv1alpha1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Update the image with the new registry and tag
		if u.GetAPIVersion() == "caching.internal.knative.dev/v1alpha1" && u.GetKind() == "Image" {
			return updateCachingImage(instance, u)
		}
		return nil
	}
}

func updateDeployment(instance *servingv1alpha1.KnativeServing, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var deployment = &appsv1.Deployment{}
	err := scheme.Scheme.Convert(u, deployment, nil)
	if err != nil {
		log.Error(err, "Error converting Unstructured to Deployment", "unstructured", u, "deployment", deployment)
		return err
	}

	registry := instance.Spec.Registry
	log.Debugw("Updating Deployment", "name", u.GetName(), "registry", registry)

	updateDeploymentImage(deployment, &registry, log)
	updatePodImagePullSecrets(deployment, &registry, log)
	err = scheme.Scheme.Convert(deployment, u, nil)
	if err != nil {
		return err
	}

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

// updateDeploymentImage updates the image of the deployment with a new registry and tag
func updateDeploymentImage(deployment *appsv1.Deployment, registry *servingv1alpha1.Registry, log *zap.SugaredLogger) {
	containers := deployment.Spec.Template.Spec.Containers
	for index := range containers {
		container := &containers[index]
		newImage := getNewImage(registry, container.Name)
		if newImage != "" {
			updateContainer(container, newImage, log)
		}
	}
	log.Debugw("Finished updating images", "name", deployment.GetName(), "containers", deployment.Spec.Template.Spec.Containers)
}

func updateCachingImage(instance *servingv1alpha1.KnativeServing, u *unstructured.Unstructured) error {
	var image = &caching.Image{}
	err := scheme.Scheme.Convert(u, image, nil)
	if err != nil {
		log.Error(err, "Error converting Unstructured to Image", "unstructured", u, "image", image)
		return err
	}

	registry := instance.Spec.Registry
	log.Debugw("Updating Image", "name", u.GetName(), "registry", registry)

	updateImageSpec(image, &registry, log)
	err = scheme.Scheme.Convert(image, u, nil)
	if err != nil {
		return err
	}
	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

// updateImageSpec updates the image of a with a new registry and tag
func updateImageSpec(image *caching.Image, registry *servingv1alpha1.Registry, log *zap.SugaredLogger) {
	newImage := getNewImage(registry, image.Name)
	if newImage != "" {
		log.Debugf("Updating image from: %v, to: %v", image.Spec.Image, newImage)
		image.Spec.Image = newImage
	}
	log.Debugw("Finished updating image", "image", image.GetName())
}

func getNewImage(registry *servingv1alpha1.Registry, containerName string) string {
	overrideImage := registry.Override[containerName]
	if overrideImage != "" {
		return overrideImage
	}
	return replaceName(registry.Default, containerName)
}

func updateContainer(container *corev1.Container, newImage string, log *zap.SugaredLogger) {
	log.Debugf("Updating container image from: %v, to: %v", container.Image, newImage)
	container.Image = newImage
}

func replaceName(imageTemplate string, name string) string {
	return strings.ReplaceAll(imageTemplate, containerNameVariable, name)
}

func updatePodImagePullSecrets(deployment *appsv1.Deployment, registry *servingv1alpha1.Registry, log *zap.SugaredLogger) {
	if len(registry.ImagePullSecrets) > 0 {
		log.Debugf("Adding ImagePullSecrets: %v", registry.ImagePullSecrets)
		deployment.Spec.Template.Spec.ImagePullSecrets = append(
			deployment.Spec.Template.Spec.ImagePullSecrets,
			registry.ImagePullSecrets...,
		)
	}
}
