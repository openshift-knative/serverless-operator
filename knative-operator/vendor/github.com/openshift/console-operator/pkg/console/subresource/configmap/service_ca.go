package configmap

import (
	corev1 "k8s.io/api/core/v1"

	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/openshift/console-operator/pkg/api"
	"github.com/openshift/console-operator/pkg/console/subresource/util"
)

const (
	// See https://github.com/openshift/service-serving-cert-signer
	injectCABundleAnnotation = "service.alpha.openshift.io/inject-cabundle"
)

// DefaultServiceCAConfigMap creates a config map that holds the service CA bundle.
// Console uses this bundle to proxy to Prometheus. The value is injected into
// key "service-ca.crt" by the service serving cert operator.
func DefaultServiceCAConfigMap(cr *operatorv1.Console) *corev1.ConfigMap {
	configMap := ServiceCAStub()
	util.AddOwnerRef(configMap, util.OwnerRefFrom(cr))
	return configMap
}

func ServiceCAStub() *corev1.ConfigMap {
	meta := util.SharedMeta()
	meta.Name = api.ServiceCAConfigMapName
	meta.Annotations = map[string]string{
		injectCABundleAnnotation: "true",
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: meta,
	}
	return configMap
}
