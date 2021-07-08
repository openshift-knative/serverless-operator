package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	KnativeUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "knative_up",
			Help: "Reports if a Knative component is up",
		},
		[]string{"type"},
	)
	KnativeServingUpG  prometheus.Gauge
	KnativeEventingUpG prometheus.Gauge
	KnativeKafkaUpG    prometheus.Gauge
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(KnativeUp)
}
