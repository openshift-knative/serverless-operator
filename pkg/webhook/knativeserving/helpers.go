package knativeserving

import (
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

// config helper to set value for key if not already set
func configure(ks *servingv1alpha1.KnativeServing, cm, key, value string) error {
	if ks.Spec.Config == nil {
		ks.Spec.Config = map[string]map[string]string{}
	}
	if len(ks.Spec.Config[cm][key]) == 0 {
		if ks.Spec.Config[cm] == nil {
			ks.Spec.Config[cm] = map[string]string{}
		}
		ks.Spec.Config[cm][key] = value
		log.Info("Configured", "map", cm, key, value)
	}
	return nil
}
