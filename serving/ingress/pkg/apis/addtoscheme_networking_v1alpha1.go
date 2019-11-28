package apis

import (
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
)

func init() {
	AddToSchemes = append(AddToSchemes, networkingv1alpha1.SchemeBuilder.AddToScheme)
}
