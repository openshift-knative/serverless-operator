package common

import (
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const MutationKey = "knative-serving-openshift/version"

var Log = logf.Log.WithName("knative").WithName("openshift")

// Configure entry in ks.Spec.Config
func Configure(ks *servingv1alpha1.KnativeServing, cm, key, value string) {
	if ks.Spec.Config == nil {
		ks.Spec.Config = map[string]map[string]string{}
	}
	if ks.Spec.Config[cm] == nil {
		ks.Spec.Config[cm] = map[string]string{}
	}
	ks.Spec.Config[cm][key] = value
	Log.Info("Configured", "map", cm, key, value)
}
