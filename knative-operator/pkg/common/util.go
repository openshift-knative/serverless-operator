package common

import (
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

const MutationTimestampKey = "knative-serving-openshift/mutation"

var Log = logf.Log.WithName("knative").WithName("openshift")

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

// updateImagesFromEnviron overrides registry images
func updateImagesFromEnviron(registry *servingv1alpha1.Registry) {
	if registry.Override == nil {
		registry.Override = map[string]string{}
	} // else return since overriding user from env might surprise me?
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], "IMAGE_") {
			name := strings.SplitN(pair[0], "_", 2)[1]
			switch name {
			case "default":
				registry.Default = pair[1]
			default:
				registry.Override[name] = pair[1]
			}
		}
	}
	log.Info("Setting", "registry", registry)
}
