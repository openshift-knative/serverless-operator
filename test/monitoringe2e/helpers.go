package monitoringe2e

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/logging"

	"github.com/openshift-knative/serverless-operator/test"

	prommodel "github.com/prometheus/common/model"
)

const (
	Interval = 10 * time.Second
)

var (
	prometheusTargetTimeout = 20 * time.Minute
	servingMetricQueries    = []string{
		"go_memory_allocations_total{job=\"activator-sm-service\"}",
		"go_memory_allocations_total{job=\"autoscaler-sm-service\"}",
		"go_memory_allocations_total{job=\"autoscaler-hpa-sm-service\"}",
		"go_memory_allocations_total{job=\"controller-sm-service\",namespace=\"knative-serving\"}",
		"go_memory_allocations_total{job=\"webhook-sm-service\"}",
	}
	eventingMetricQueries = []string{
		"go_memory_allocations_total{job=\"eventing-controller-sm-service\",namespace=\"knative-eventing\"}",
		"go_memory_allocations_total{job=\"eventing-webhook-sm-service\"}",
		"go_memory_allocations_total{job=\"imc-controller-sm-service\"}",
		// TODO: imc-dispatcher-sm-service removed due to upstream bug - metrics endpoint fails with duplicate metric registration
		// "go_memory_allocations_total{job=\"imc-dispatcher-sm-service\"}",
		"go_memory_allocations_total{job=\"mt-broker-controller-sm-service\"}",
		"go_memory_allocations_total{job=\"mt-broker-filter-sm-service\"}",
		"go_memory_allocations_total{job=\"mt-broker-ingress-sm-service\"}",
	}

	EventingPingSourceMetricQueries = []string{
		"go_memory_allocations_total{job=\"pingsource-mt-adapter-sm-service\"}",
	}

	KafkaQueries = []string{
		"go_memory_allocations_total{job=\"kafka-controller-sm-service\"}",
		"go_memory_allocations_total{job=\"kafka-webhook-eventing-sm-service\"}",
	}

	KafkaBrokerDataPlaneQueries = []string{
		"sum(rate(kn_eventing_dispatch_latency_milliseconds_bucket{le=\"100.0\", namespace=\"knative-eventing\", job=\"kafka-broker-receiver-sm-service\"}[5m])) by (name, namespace_name) / sum(rate(kn_eventing_dispatch_latency_milliseconds_count{job=\"kafka-broker-receiver-sm-service\", namespace=\"knative-eventing\",}[5m])) by (name, namespace_name)",
		"sum(rate(kn_eventing_dispatch_latency_milliseconds_bucket{le=\"100.0\", job=\"kafka-broker-dispatcher-sm-service\", namespace=\"knative-eventing\"}[5m])) by (name, namespace_name) / sum(rate(kn_eventing_dispatch_latency_milliseconds_count{job=\"kafka-broker-dispatcher-sm-service\", namespace=\"knative-eventing\"}[5m])) by (name, namespace_name)",
		"sum(http_events_sent_total{job=\"kafka-broker-receiver-sm-service\", namespace=\"knative-eventing\"}) by (name, namespace_name)",
		"sum(http_events_sent_total{job=\"kafka-broker-dispatcher-sm-service\", namespace=\"knative-eventing\"}) by (name, namespace_name)",
	}

	NamespacedKafkaBrokerDataPlaneQueries = func(namespace string) []string {
		return []string{
			fmt.Sprintf("sum(rate(kn_eventing_dispatch_latency_milliseconds_bucket{le=\"100.0\", namespace=\"%s\", job=\"kafka-broker-receiver-sm-service\"}[5m])) by (name, namespace_name) / sum(rate(kn_eventing_dispatch_latency_milliseconds_count{job=\"kafka-broker-receiver-sm-service\", namespace=\"%s\",}[5m])) by (name, namespace_name)", namespace, namespace),
			fmt.Sprintf("sum(rate(kn_eventing_dispatch_latency_milliseconds_bucket{le=\"100.0\", job=\"kafka-broker-dispatcher-sm-service\", namespace=\"%s\"}[5m])) by (name, namespace_name) / sum(rate(kn_eventing_dispatch_latency_milliseconds_count{job=\"kafka-broker-dispatcher-sm-service\", namespace=\"%s\"}[5m])) by (name, namespace_name)", namespace, namespace),
			fmt.Sprintf("sum(http_events_sent_total{job=\"kafka-broker-receiver-sm-service\", namespace=\"%s\"}) by (name, namespace_name)", namespace),
			fmt.Sprintf("sum(http_events_sent_total{job=\"kafka-broker-dispatcher-sm-service\", namespace=\"%s\"}) by (name, namespace_name)", namespace),
		}
	}

	KafkaControllerQueries = []string{
		"sum(rate(http_client_request_duration_seconds_bucket{le=~\"0.1\", job=\"kafka-controller-sm-service\", namespace=\"knative-eventing\"}[5m])) / sum(rate(http_client_request_duration_seconds_count{job=\"kafka-controller-sm-service\", namespace=\"knative-eventing\"}[5m]))",
		"sum(kn_workqueue_unfinished_work_seconds{job=\"kafka-controller-sm-service\", namespace=\"knative-eventing\"}) by (name)",
	}

	serverlessComponentQueries = []string{
		// Checks if openshift-knative-operator metrics are served
		"go_memory_allocations_total{job=\"knative-operator-metrics\"}",
		// Checks if knative-openshift metrics are served
		"controller_runtime_active_workers{controller=\"knativeserving-controller\"}",
		// Checks if knative-openshift-ingress metrics are served
		"go_memory_allocations_total{job=\"knative-openshift-ingress-metrics\"}",
	}
)

func VerifyHealthStatusMetric(ctx context.Context, label string, expectedValue string) error {
	pc, err := test.NewPrometheusClient(ctx)
	if err != nil {
		return err
	}

	if err := wait.PollUntilContextTimeout(ctx, Interval, prometheusTargetTimeout, true, func(_ context.Context) (bool, error) {
		value, _, err := pc.Query(context.Background(), fmt.Sprintf(`knative_up{type="%s"}`, label), time.Time{})
		if err != nil {
			logging.FromContext(ctx).Info("Error querying prometheus metrics:", err)
			return false, nil
		}

		vec, ok := value.(prommodel.Vector)
		if !ok {
			return false, nil
		}

		if len(vec) < 1 {
			return false, nil
		}

		logging.FromContext(ctx).Infof("Vector value: %v", vec[0].Value.String())
		return vec[0].Value.String() == expectedValue, nil
	}); err != nil {
		return fmt.Errorf("failed to access the Prometheus API endpoint and get the metric value expected: %w", err)
	}
	return nil
}

func VerifyMetrics(ctx context.Context, metricQueries []string) error {
	pc, err := test.NewPrometheusClient(ctx)
	if err != nil {
		return err
	}

	for _, metric := range metricQueries {
		if err := wait.PollUntilContextTimeout(ctx, Interval, prometheusTargetTimeout, true, func(_ context.Context) (bool, error) {
			value, _, err := pc.Query(context.Background(), metric, time.Time{})
			if err != nil {
				logging.FromContext(ctx).Info("Error querying prometheus metrics:", err)
				return false, nil
			}

			if value.Type() != prommodel.ValVector {
				return false, nil
			}

			vector := value.(prommodel.Vector)
			return vector.Len() > 0, nil
		}); err != nil {
			return fmt.Errorf("failed to access the Prometheus API endpoint for %s and get the metric value expected: %w", metric, err)
		}
	}
	return nil
}
