package apis

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

func init() {
	AddToSchemes = append(AddToSchemes, monitoringv1.AddToScheme)
}
