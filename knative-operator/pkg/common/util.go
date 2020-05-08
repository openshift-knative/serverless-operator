package common

import (
	"fmt"
	"strings"

	servingv1alpha1 "github.com/knative-sandbox/operator/pkg/apis/operator/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

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

// BuildImageOverrideMapFromEnviron creates a map to overrides registry images
func BuildImageOverrideMapFromEnviron(environ []string) map[string]string {
	overrideMap := map[string]string{}

	for _, e := range environ {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], "IMAGE_") {
			// convert
			// "IMAGE_container=docker.io/foo"
			// "IMAGE_deployment_container=docker.io/foo2"
			// to
			// container: docker.io/foo
			// deployment/container: docker.io/foo2
			var name string
			parts := strings.SplitN(pair[0], "_", -1)
			if len(parts) == 3 {
				name = fmt.Sprintf("%s/%s", parts[1], parts[2])
			} else {
				name = parts[1]
			}

			if pair[1] != "" {
				overrideMap[name] = pair[1]
			}
		}
	}
	return overrideMap
}
