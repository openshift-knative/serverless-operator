package serving

import (
	common "github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

const MutationTimestampKey = "knative-serving-openshift/mutation"

var Log = common.Log.WithName("serving")

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
