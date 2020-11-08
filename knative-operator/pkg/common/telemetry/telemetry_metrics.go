package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	serverlessTelemetryG = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "serverless_telemetry",
			Help: "Reports number of serverless resources for telemetry",
		},
		[]string{"type"},
	)
	servicesG  prometheus.Gauge
	routesG    prometheus.Gauge
	revisionsG prometheus.Gauge
	sourcesG   prometheus.Gauge
)

func init() {
	// Register custom telemetry metrics with the global prometheus registry
	metrics.Registry.MustRegister(serverlessTelemetryG)
}
