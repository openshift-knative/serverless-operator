package knativekafka

import (
	"github.com/go-logr/logr"
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

var delimiter = "/"

// ImageTransformer is an interface for transforming images passed to the ResourceImageTransformer
type ImageTransformer interface {
	ImageForContainer(container *corev1.Container, parentName string) (string, bool)
	ImageForEnvVar(env *corev1.EnvVar, parentName string) (string, bool)
}

// registryImageTransformer is a v1alpha1.Registry specific transformer
type registryImageTransformer struct {
	overrideMap map[string]string
}

var _ ImageTransformer = (*registryImageTransformer)(nil)

func (rit *registryImageTransformer) ImageForContainer(container *corev1.Container, parentName string) (string, bool) {
	return rit.handleImage(container.Name, parentName)
}

func (rit *registryImageTransformer) ImageForEnvVar(env *corev1.EnvVar, parentName string) (string, bool) {
	return rit.handleImage(env.Name, "")
}

func (rit *registryImageTransformer) handleImage(resourceName, parentName string) (string, bool) {
	if image, ok := rit.overrideMap[parentName+delimiter+resourceName]; ok {
		return image, true
	}
	if image, ok := rit.overrideMap[resourceName]; ok {
		return image, true
	}
	return "", false
}

// ImageTransform updates image with a new registry and tag
func ImageTransform(overrideMap map[string]string, log logr.Logger) mf.Transformer {
	rit := &registryImageTransformer{
		overrideMap: overrideMap,
	}
	return ResourceImageTransformer(rit, log)
}

// ResourceImageTransformer takes an ImageTransformer and transform images across resources
func ResourceImageTransformer(imageTransformer ImageTransformer, log logr.Logger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		switch u.GetKind() {
		// TODO need to use PodSpecable duck type in order to remove duplicates of deployment, daemonSet
		case "Deployment":
			return updateDeployment(imageTransformer, u, log)
		case "DaemonSet":
			return updateDaemonSet(imageTransformer, u, log)
		case "Job":
			return updateJob(imageTransformer, u, log)
		}
		return nil
	}
}

func updateDeployment(imageTransformer ImageTransformer, u *unstructured.Unstructured, log logr.Logger) error {
	var deployment = &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
		log.Error(err, "Error converting Unstructured to Deployment", "unstructured", u, "deployment", deployment)
		return err
	}

	updateRegistry(&deployment.Spec.Template.Spec, imageTransformer, log, deployment.GetName())
	if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Info("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateDaemonSet(imageTransformer ImageTransformer, u *unstructured.Unstructured, log logr.Logger) error {
	var daemonSet = &appsv1.DaemonSet{}
	if err := scheme.Scheme.Convert(u, daemonSet, nil); err != nil {
		log.Error(err, "Error converting Unstructured to daemonSet", "unstructured", u, "daemonSet", daemonSet)
		return err
	}
	updateRegistry(&daemonSet.Spec.Template.Spec, imageTransformer, log, daemonSet.GetName())
	if err := scheme.Scheme.Convert(daemonSet, u, nil); err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Info("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateJob(imageTransformer ImageTransformer, u *unstructured.Unstructured, log logr.Logger) error {
	var job = &batchv1.Job{}
	if err := scheme.Scheme.Convert(u, job, nil); err != nil {
		log.Error(err, "Error converting Unstructured to job", "unstructured", u, "job", job)
		return err
	}
	updateRegistry(&job.Spec.Template.Spec, imageTransformer, log, job.GetName())
	if err := scheme.Scheme.Convert(job, u, nil); err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Info("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateRegistry(spec *corev1.PodSpec, imageTransformer ImageTransformer, log logr.Logger, name string) {
	log.Info("Updating", "name", name, "imageTransformer", imageTransformer)

	updateImage(spec, imageTransformer, log, name)
	updateEnvVarImages(spec, imageTransformer)
}

// updateImage updates the image with a new registry and tag
func updateImage(spec *corev1.PodSpec, imageTransformer ImageTransformer, log logr.Logger, name string) {
	containers := spec.Containers
	for index := range containers {
		container := &containers[index]
		newImage, _ := imageTransformer.ImageForContainer(container, name)
		if newImage != "" {
			updateContainer(container, newImage, log)
		}
	}
	log.Info("Finished updating images", "name", name, "containers", spec.Containers)
}

func updateEnvVarImages(spec *corev1.PodSpec, imageTransformer ImageTransformer) {
	containers := spec.Containers
	for index := range containers {
		container := &containers[index]
		for envIndex := range container.Env {
			env := &container.Env[envIndex]
			if newImage, ok := imageTransformer.ImageForEnvVar(env, container.Name); ok {
				env.Value = newImage
			}
		}
	}
}

func updateContainer(container *corev1.Container, newImage string, log logr.Logger) {
	log.Info("Updating container image from: %v, to: %v", container.Image, newImage)
	container.Image = newImage
}
