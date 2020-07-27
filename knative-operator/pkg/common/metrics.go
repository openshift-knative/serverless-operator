package common

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	KnativeServingReadyG = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "knative_serving_ready",
			Help: "Reports if Knative Serving is up",
		},
	)
	KnativeEventingReadyG = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "knative_eventing_ready",
			Help: "Reports if Knative Eventing is up",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(KnativeServingReadyG, KnativeEventingReadyG)
}
