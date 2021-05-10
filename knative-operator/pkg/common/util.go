package common

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var Log = logf.Log.WithName("knative").WithName("openshift")

// Configure is a  helper to set a value for a key, potentially overriding existing contents.
func Configure(ks *operatorv1alpha1.KnativeServing, cm, key, value string) bool {
	if ks.Spec.Config == nil {
		ks.Spec.Config = map[string]map[string]string{}
	}

	old, found := ks.Spec.Config[cm][key]
	if found && value == old {
		return false
	}

	if ks.Spec.Config[cm] == nil {
		ks.Spec.Config[cm] = map[string]string{}
	}

	ks.Spec.Config[cm][key] = value
	Log.Info("Configured", "map", cm, key, value, "old value", old)
	return true
}

// BuildImageOverrideMapFromEnviron creates a map to overrides registry images
func BuildImageOverrideMapFromEnviron(environ []string, prefix string) map[string]string {
	overrideMap := map[string]string{}

	for _, e := range environ {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], prefix) {
			// convert
			// "IMAGE_container=docker.io/foo"
			// "IMAGE_deployment__container=docker.io/foo2"
			// "IMAGE_env_var=docker.io/foo3"
			// "IMAGE_deployment__env_var=docker.io/foo4"
			// to
			// container: docker.io/foo
			// deployment/container: docker.io/foo2
			// env_var: docker.io/foo3
			// deployment/env_var: docker.io/foo4
			name := strings.TrimPrefix(pair[0], prefix)
			name = strings.Replace(name, "__", "/", 1)
			if pair[1] != "" {
				overrideMap[name] = pair[1]
			}
		}
	}
	return overrideMap
}

// SetAnnotations is a transformer to set annotations on given object
// The existing annotations are kept as is, except they are overridden with the
// annotations given as the argument.
func SetAnnotations(annotations map[string]string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetAnnotations() == nil {
			u.SetAnnotations(annotations)
		} else {
			res := u.GetAnnotations()
			for key, value := range annotations {
				res[key] = value
			}
			u.SetAnnotations(res)
		}
		return nil
	}
}

func EnsureContainerMemoryLimit(s *operatorv1alpha1.CommonSpec, containerName string, memory resource.Quantity) {
	for i, v := range s.Resources {
		if v.Container == containerName {
			if v.Limits == nil {
				v.Limits = corev1.ResourceList{}
			}
			if _, ok := v.Limits[corev1.ResourceMemory]; ok {
				return
			}
			v.Limits[corev1.ResourceMemory] = memory
			s.Resources[i] = v
			return
		}
	}
	s.Resources = append(s.Resources, operatorv1alpha1.ResourceRequirementsOverride{
		Container: containerName,
		ResourceRequirements: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: memory,
			},
		},
	})
}

// common function to enqueue reconcile requests for resources
func EnqueueRequestByOwnerAnnotations(ownerNameAnnotationKey, ownerNamespaceAnnotationKey string) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
		annotations := obj.GetAnnotations()
		ownerNamespace := annotations[ownerNamespaceAnnotationKey]
		ownerName := annotations[ownerNameAnnotationKey]
		if ownerNamespace != "" && ownerName != "" {
			return []reconcile.Request{{
				NamespacedName: types.NamespacedName{Namespace: ownerNamespace, Name: ownerName},
			}}
		}
		return nil
	})
}

func BuildGVKToResourceMap(manifests ...mf.Manifest) map[schema.GroupVersionKind]client.Object {
	gvkToResource := make(map[schema.GroupVersionKind]client.Object)

	for _, manifest := range manifests {
		resources := manifest.Resources()

		for i := range resources {
			// it is ok to overwrite existing since we are only interested
			// in the types of the resources, not the instances
			gvkToResource[resources[i].GroupVersionKind()] = &resources[i]
		}
	}

	return gvkToResource
}
