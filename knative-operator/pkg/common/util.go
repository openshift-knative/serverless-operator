package common

import (
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var Log = logf.Log.WithName("knative").WithName("openshift")

// StringMap is a map which key and value are strings
type StringMap map[string]string

// Removes given slice from StringMap
func (m StringMap) Remove(toRemove string) StringMap {
	delete(m, toRemove)
	return m
}

// Gets StringMap values as comma separated string
func (m StringMap) StringValues() string {
	values := make([]string, 0, len(m))

	for _, v := range m {
		values = append(values, v)
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

// Configure is a  helper to set a value for a key, potentially overriding existing contents.
func Configure(ks *operatorv1beta1.KnativeServing, cm, key, value string) bool {
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
			// "IMAGE_container=quay.io/foo"
			// "IMAGE_deployment__container=quay.io/foo2"
			// "IMAGE_env_var=quay.io/foo3"
			// "IMAGE_deployment__env_var=quay.io/foo4"
			// to
			// container: quay.io/foo
			// deployment/container: quay.io/foo2
			// env_var: quay.io/foo3
			// deployment/env_var: quay.io/foo4
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

// EnqueueRequestByOwnerAnnotations is a common function to enqueue reconcile requests for resources.
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

type SkipPredicate struct {
	predicate.Funcs
}

func (SkipPredicate) Delete(e event.DeleteEvent) bool {
	return false
}
