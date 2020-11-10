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
	serviceG           = serverlessTelemetryG.WithLabelValues("service")
	routeG             = serverlessTelemetryG.WithLabelValues("route")
	revisionG          = serverlessTelemetryG.WithLabelValues("revision")
	configurationG     = serverlessTelemetryG.WithLabelValues("configuration")
	pingSourceG        = serverlessTelemetryG.WithLabelValues("source_ping")
	apiServerSourceG   = serverlessTelemetryG.WithLabelValues("source_apiserver")
	sinkBindingSourceG = serverlessTelemetryG.WithLabelValues("source_sinkbinding")
	kafkaSourceG       = serverlessTelemetryG.WithLabelValues("source_kafka")
)

func init() {
	// Register custom telemetry metrics with the global prometheus registry
	metrics.Registry.MustRegister(serverlessTelemetryG)
}
