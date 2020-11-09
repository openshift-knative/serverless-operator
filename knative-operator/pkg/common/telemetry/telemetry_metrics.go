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
	serviceG           prometheus.Gauge
	routeG             prometheus.Gauge
	revisionG          prometheus.Gauge
	configurationG     prometheus.Gauge
	pingSourceG        prometheus.Gauge
	apiServerSourceG   prometheus.Gauge
	sinkBindingSourceG prometheus.Gauge
	kafkaSourceG       prometheus.Gauge
)

func init() {
	// Register custom telemetry metrics with the global prometheus registry
	metrics.Registry.MustRegister(serverlessTelemetryG)
}
