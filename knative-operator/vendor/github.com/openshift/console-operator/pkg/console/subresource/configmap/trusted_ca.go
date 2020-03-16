package configmap

import (
	corev1 "k8s.io/api/core/v1"

	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/openshift/console-operator/pkg/api"
	"github.com/openshift/console-operator/pkg/console/subresource/util"
)

const (
	injectTrustedCABundleLabel = "config.openshift.io/inject-trusted-cabundle"
)

func DefaultTrustedCAConfigMap(cr *operatorv1.Console) *corev1.ConfigMap {
	configMap := TrustedCAStub()
	util.AddOwnerRef(configMap, util.OwnerRefFrom(cr))
	return configMap
}

func TrustedCAStub() *corev1.ConfigMap {
	meta := util.SharedMeta()
	meta.Name = api.TrustedCAConfigMapName
	meta.Labels = map[string]string{
		injectTrustedCABundleLabel: "true",
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: meta,
		Data: map[string]string{
			api.TrustedCABundleKey: "",
		},
	}
	return configMap
}
