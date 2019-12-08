package pkg

import (
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var Log = logf.Log.WithName("knative").WithName("openshift")

// config helper to set value for key if not already set
func Configure(ks *servingv1alpha1.KnativeServing, cm, key, value string) bool {
	if ks.Spec.Config == nil {
		ks.Spec.Config = map[string]map[string]string{}
	}
	if _, found := ks.Spec.Config[cm][key]; !found {
		if ks.Spec.Config[cm] == nil {
			ks.Spec.Config[cm] = map[string]string{}
		}
		ks.Spec.Config[cm][key] = value
		Log.Info("Configured", "map", cm, key, value)
		return true
	}
	return false
}
