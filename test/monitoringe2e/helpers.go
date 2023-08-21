package monitoringe2e

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/logging"

	prommodel "github.com/prometheus/common/model"
)

const (
	Interval = 10 * time.Second
)

var (
	prometheusTargetTimeout = 20 * time.Minute
	servingMetricQueries    = []string{
		"activator_go_mallocs",
		"autoscaler_go_mallocs",
		"hpaautoscaler_go_mallocs",
		"controller_go_mallocs{namespace=\"knative-serving\"}",
		"domainmapping_go_mallocs",
		"domainmapping_webhook_go_mallocs",
		"webhook_go_mallocs",
	}
	eventingMetricQueries = []string{
		"controller_go_mallocs{namespace=\"knative-eventing\"}",
		"eventing_webhook_go_mallocs",
		"inmemorychannel_webhook_go_mallocs",
		"inmemorychannel_dispatcher_go_mallocs",
		"mt_broker_controller_go_mallocs",
		"mt_broker_filter_go_mallocs",
		"mt_broker_ingress_go_mallocs",
	}

	KafkaQueries = []string{
		"kafka_broker_controller_go_mallocs",
		"kafka_webhook_eventing_go_mallocs",
	}

	KafkaBrokerDataPlaneQueries = []string{
		"sum(rate(event_dispatch_latencies_ms_bucket{le=\"100.0\", namespace=\"knative-eventing\", job=\"kafka-broker-receiver-sm-service\"}[5m])) by (name, namespace_name) / sum(rate(event_dispatch_latencies_ms_count{job=\"kafka-broker-receiver-sm-service\", namespace=\"knative-eventing\",}[5m])) by (name, namespace_name)",
		"sum(rate(event_dispatch_latencies_ms_bucket{le=\"100.0\", job=\"kafka-broker-dispatcher-sm-service\", namespace=\"knative-eventing\"}[5m])) by (name, namespace_name) / sum(rate(event_dispatch_latencies_ms_count{job=\"kafka-broker-dispatcher-sm-service\", namespace=\"knative-eventing\"}[5m])) by (name, namespace_name)",
		"sum(event_count_1_total{job=\"kafka-broker-receiver-sm-service\", namespace=\"knative-eventing\"}) by (name, namespace_name)",
		"sum(event_count_1_total{job=\"kafka-broker-dispatcher-sm-service\", namespace=\"knative-eventing\"}) by (name, namespace_name)",
	}

	NamespacedKafkaBrokerDataPlaneQueries = func(namespace string) []string {
		return []string{
			fmt.Sprintf("sum(rate(event_dispatch_latencies_ms_bucket{le=\"100.0\", namespace=\"%s\", job=\"kafka-broker-receiver-sm-service\"}[5m])) by (name, namespace_name) / sum(rate(event_dispatch_latencies_ms_count{job=\"kafka-broker-receiver-sm-service\", namespace=\"%s\",}[5m])) by (name, namespace_name)", namespace, namespace),
			fmt.Sprintf("sum(rate(event_dispatch_latencies_ms_bucket{le=\"100.0\", job=\"kafka-broker-dispatcher-sm-service\", namespace=\"%s\"}[5m])) by (name, namespace_name) / sum(rate(event_dispatch_latencies_ms_count{job=\"kafka-broker-dispatcher-sm-service\", namespace=\"%s\"}[5m])) by (name, namespace_name)", namespace, namespace),
			fmt.Sprintf("sum(event_count_1_total{job=\"kafka-broker-receiver-sm-service\", namespace=\"%s\"}) by (name, namespace_name)", namespace),
			fmt.Sprintf("sum(event_count_1_total{job=\"kafka-broker-dispatcher-sm-service\", namespace=\"%s\"}) by (name, namespace_name)", namespace),
		}
	}

	KafkaControllerQueries = []string{
		"sum(rate(kafka_broker_controller_reconcile_latency_bucket{le=\"100\", job=\"kafka-controller-sm-service\", namespace=\"knative-eventing\"}[5m])) / sum(rate(kafka_broker_controller_reconcile_latency_count{job=\"kafka-controller-sm-service\", namespace=\"knative-eventing\"}[5m]))",
		"sum(kafka_broker_controller_workqueue_depth{job=\"kafka-controller-sm-service\", namespace=\"knative-eventing\"}) by (name)",
	}

	serverlessComponentQueries = []string{
		// Checks if openshift-knative-operator metrics are served
		"knative_operator_go_mallocs",
		// Checks if knative-openshift metrics are served
		"controller_runtime_active_workers{controller=\"knativeserving-controller\"}",
		// Checks if knative-openshift-ingress metrics are served
		"openshift_ingress_controller_go_mallocs",
	}
)

func VerifyHealthStatusMetric(ctx context.Context, label string, expectedValue string) error {
	pc, err := test.NewPrometheusClient(ctx)
	if err != nil {
		return err
	}

	if err := wait.PollImmediate(Interval, prometheusTargetTimeout, func() (bool, error) {
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
		if err := wait.PollImmediate(Interval, prometheusTargetTimeout, func() (bool, error) {
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
