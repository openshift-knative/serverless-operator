package common

import (
	"strings"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var Log = logf.Log.WithName("knative").WithName("openshift")

const ImagePrefix = "IMAGE_"

// Configure is a  helper to set a value for a key, potentially overriding existing contents.
func Configure(ks *servingv1alpha1.KnativeServing, cm, key, value string) bool {
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

// IngressNamespace returns namespace where ingress is deployed.
func IngressNamespace(servingNamespace string) string {
	return servingNamespace + "-ingress"
}

// BuildImageOverrideMapFromEnviron creates a map to overrides registry images
func BuildImageOverrideMapFromEnviron(environ []string) map[string]string {
	overrideMap := map[string]string{}

	for _, e := range environ {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], ImagePrefix) {
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
			name := strings.TrimPrefix(pair[0], ImagePrefix)
			name = strings.Replace(name, "__", "/", 1)
			if pair[1] != "" {
				overrideMap[name] = pair[1]
			}
		}
	}
	return overrideMap
}

// SetOwnerAnnotations is a transformer to set owner annotations on given object
func SetOwnerAnnotations(instance *servingv1alpha1.KnativeServing) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		u.SetAnnotations(map[string]string{
			ServingOwnerName:      instance.Name,
			ServingOwnerNamespace: instance.Namespace,
		})
		return nil
	}
}
