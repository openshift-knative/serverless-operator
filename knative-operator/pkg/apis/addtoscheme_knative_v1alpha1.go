package apis

import (
	"github.com/knative-sandbox/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/apis/istio/v1alpha3"
)

func init() {
	AddToSchemes = append(AddToSchemes, v1alpha1.AddToScheme, v1alpha3.AddToScheme)
}
