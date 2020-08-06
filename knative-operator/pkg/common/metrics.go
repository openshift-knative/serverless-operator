package common

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	KnativeServingUpG = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "knative_up",
			Help:        "Reports if a Knative component is up",
			ConstLabels: map[string]string{"type": "serving_status"},
		},
	)
	KnativeEventingUpG = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "knative_up",
			Help:        "Reports if a Knative component is up",
			ConstLabels: map[string]string{"type": "eventing_status"},
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(KnativeServingUpG, KnativeEventingUpG)
}
