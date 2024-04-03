package common

import (
	"context"
	"fmt"
	"sort"
	"strings"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/configmap"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
)

// InjectEnvironmentIntoDeployment injects the specified environment variables into the
// specified deployment/container.
// Note: We're not deleting empty environment variables and instead set them to empty
// string. Three-way-merging of the deployment drops the update otherwise.
func InjectEnvironmentIntoDeployment(deploymentName, containerName string, envs ...corev1.EnvVar) mf.Transformer {
	return transformDeployment(deploymentName, func(deploy *appsv1.Deployment) error {
		containers := deploy.Spec.Template.Spec.Containers
		for i := range containers {
			c := &containers[i]
			if c.Name != containerName {
				continue
			}

			for _, val := range envs {
				c.Env = upsert(c.Env, val)
			}
		}

		return nil
	})
}

// upsert updates the env var if the key already exists or inserts it if it didn't
// exist.
func upsert(orgEnv []corev1.EnvVar, val corev1.EnvVar) []corev1.EnvVar {
	// Set the value if the key is already present.
	for i := range orgEnv {
		if orgEnv[i].Name == val.Name {
			orgEnv[i].Value = val.Value
			return orgEnv
		}
	}
	// If not, append a key-value pair.
	return append(orgEnv, val)
}

// transformDeployment returns a transformer that transforms a deployment with the given
// name.
func transformDeployment(name string, f func(*appsv1.Deployment) error) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Deployment" || u.GetName() != name {
			return nil
		}

		deployment := &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
			return err
		}

		if err := f(deployment); err != nil {
			return err
		}

		return scheme.Scheme.Convert(deployment, u, nil)
	}
}

func ConfigMapVolumeChecksumTransform(ctx context.Context, c client.Client, configMaps sets.Set[string]) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		namespace := u.GetNamespace()
		var podSpec *corev1.PodTemplateSpec
		var obj runtime.Object
		if u.GetKind() == "Deployment" {
			d := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, d, nil); err != nil {
				return err
			}
			podSpec = &d.Spec.Template
			obj = d
		}
		if u.GetKind() == "StatefulSet" {
			ss := &appsv1.StatefulSet{}
			if err := scheme.Scheme.Convert(u, ss, nil); err != nil {
				return err
			}
			podSpec = &ss.Spec.Template
			obj = ss
		}

		if podSpec == nil {
			return nil
		}

		configMaps := configMaps.Clone()

		// we need to have a stable algorithm since Go maps aren't sorted or traversed always in the same order
		// we use a sorted array of key+value elements
		var kvs []string

		for _, v := range podSpec.Spec.Volumes {
			if v.ConfigMap != nil && configMaps.Has(v.ConfigMap.Name) {
				cm := &corev1.ConfigMap{}
				err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: v.ConfigMap.Name}, cm)
				if apierrors.IsNotFound(err) {
					continue
				}
				if err != nil {
					return fmt.Errorf("failed to get ConfigMap %s/%s: %w", namespace, v.ConfigMap.Name, err)
				}

				configMaps.Delete(cm.GetName())

				for k, v := range cm.Data {
					kvs = append(kvs, k+v)
				}
				for k, v := range cm.BinaryData {
					kvs = append(kvs, k+string(v))
				}

			}
		}
		for _, name := range sets.List(configMaps) {
			cm := &corev1.ConfigMap{}
			err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, cm)
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to get ConfigMap %s/%s: %w", namespace, name, err)
			}

			for k, v := range cm.Data {
				kvs = append(kvs, k+v)
			}
			for k, v := range cm.BinaryData {
				kvs = append(kvs, k+string(v))
			}

		}

		sort.Strings(kvs)
		checksum := configmap.Checksum(strings.Join(kvs, ""))

		if podSpec.Annotations == nil {
			podSpec.Annotations = make(map[string]string, 1)
		}
		podSpec.Annotations[common.VolumeChecksumAnnotation] = checksum

		return scheme.Scheme.Convert(obj, u, nil)
	}
}
