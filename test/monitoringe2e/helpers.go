package monitoringe2e

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift-knative/serverless-operator/test"

	prom "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
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

type authRoundtripper struct {
	authorization string
	inner         http.RoundTripper
}

func (a *authRoundtripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", a.authorization)
	return a.inner.RoundTrip(r)
}

func newPrometheusClient(caCtx *test.Context) (promv1.API, error) {
	route, err := getPrometheusRoute(caCtx)
	if err != nil {
		return nil, err
	}
	bToken, err := getBearerTokenForPrometheusAccount(caCtx)
	if err != nil {
		return nil, err
	}

	rt := prom.DefaultRoundTripper.(*http.Transport).Clone()
	rt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client, err := prom.NewClient(prom.Config{
		Address: "https://" + route.Spec.Host,
		RoundTripper: &authRoundtripper{
			authorization: fmt.Sprintf("Bearer %s", bToken),
			inner:         rt,
		},
	})
	if err != nil {
		return nil, err
	}

	return promv1.NewAPI(client), nil
}

func getPrometheusRoute(caCtx *test.Context) (*routev1.Route, error) {
	r, err := caCtx.Clients.Route.Routes("openshift-monitoring").Get(context.Background(), "prometheus-k8s", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting Prometheus route: %w", err)
	}
	return r, nil
}

func VerifyHealthStatusMetric(caCtx *test.Context, label string, expectedValue string) error {
	pc, err := newPrometheusClient(caCtx)
	if err != nil {
		return err
	}

	if err := wait.PollImmediate(test.Interval, prometheusTargetTimeout, func() (bool, error) {
		value, _, err := pc.Query(context.Background(), fmt.Sprintf(`knative_up{type="%s"}`, label), time.Time{})
		if err != nil {
			caCtx.T.Log("Error querying prometheus metrics:", err)
			return false, nil
		}

		vec, ok := value.(prommodel.Vector)
		if !ok {
			return false, nil
		}

		if len(vec) < 1 {
			return false, nil
		}

		caCtx.T.Logf("Vector value: %v", vec[0].Value.String())
		return vec[0].Value.String() == expectedValue, nil
	}); err != nil {
		return fmt.Errorf("failed to access the Prometheus API endpoint and get the metric value expected: %w", err)
	}
	return nil
}

func VerifyMetrics(caCtx *test.Context, metricQueries []string) error {
	pc, err := newPrometheusClient(caCtx)
	if err != nil {
		return err
	}

	for _, metric := range metricQueries {
		if err := wait.PollImmediate(test.Interval, prometheusTargetTimeout, func() (bool, error) {
			value, _, err := pc.Query(context.Background(), metric, time.Time{})
			if err != nil {
				caCtx.T.Log("Error querying prometheus metrics:", err)
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

func getBearerTokenForPrometheusAccount(caCtx *test.Context) (string, error) {
	secrets, err := caCtx.Clients.Kube.CoreV1().Secrets("openshift-monitoring").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error listing secrets in namespace openshift-monitoring: %w", err)
	}
	tokenSecret := getSecretNameForToken(secrets.Items)
	if tokenSecret == "" {
		return "", errors.New("token name for prometheus-k8s service account not found")
	}
	sec, err := caCtx.Clients.Kube.CoreV1().Secrets("openshift-monitoring").Get(context.Background(), tokenSecret, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting secret %s: %w", tokenSecret, err)
	}
	tokenContents := sec.Data["token"]
	if len(tokenContents) == 0 {
		return "", fmt.Errorf("token data is missing for token %s", tokenSecret)
	}
	return string(tokenContents), nil
}

func getSecretNameForToken(secrets []corev1.Secret) string {
	for _, sec := range secrets {
		if strings.HasPrefix(sec.Name, "prometheus-k8s-token") {
			return sec.Name
		}
	}
	return ""
}
