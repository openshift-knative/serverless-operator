package e2e

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Jeffail/gabs"
	"github.com/openshift-knative/serverless-operator/test"
	v1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/spoof"
)

const servingNamespace = "knative-serving"

type PrometheusInfo struct {
	options   []interface{}
	queryPath string
	token     string
}

func NewPrometheusInfo(caCtx *test.Context) *PrometheusInfo {
	route := getPrometheusRoute(caCtx)
	bToken := getBearerTokenForPrometheusAccount(caCtx)
	var reqOption pkgTest.RequestOption = func(request *http.Request) {
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bToken))
	}
	var transportOption spoof.TransportOption = func(transport *http.Transport) *http.Transport {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return transport
	}
	pc := &PrometheusInfo{
		options:   []interface{}{reqOption, transportOption},
		queryPath: "https://" + route.Spec.Host + "/api/v1/query?query=",
		token:     bToken,
	}
	return pc
}

func getPrometheusRoute(caCtx *test.Context) *v1.Route {
	r, err := caCtx.Clients.Route.Routes("openshift-monitoring").Get(context.Background(), "prometheus-k8s", meta.GetOptions{})
	if err != nil {
		caCtx.T.Fatalf("Error getting Prometheus route %v", err)
	}
	return r
}

func VerifyHealthStatusMetric(caCtx *test.Context, label string, expectedValue string) {
	pc := NewPrometheusInfo(caCtx)
	path := fmt.Sprintf("%s%s", pc.queryPath, url.QueryEscape(fmt.Sprintf(`knative_up{type="%s"}`, label)))
	metricsURL, err := url.Parse(path)
	if err != nil {
		caCtx.T.Fatalf("Error parsing url for metrics: %v", err)
	}
	// Wait until the endpoint is actually working and we get the expected value back
	_, err = pkgTest.WaitForEndpointState(
		context.Background(),
		caCtx.Clients.Kube,
		caCtx.T.Logf,
		metricsURL,
		eventuallyMatchesValue(expectedValue),
		"WaitForMetricsToServeText",
		true, pc.options...)
	if err != nil {
		caCtx.T.Fatalf("Failed to access the Prometheus API endpoint and get the metric value expected: %v", err)
	}
}

func VerifyServingControlPlaneMetrics(caCtx *test.Context) {
	pc := NewPrometheusInfo(caCtx)
	servingMetrics := []string{
		"activator_client_results",
		"autoscaler_actual_pods",
		"hpaautoscaler_client_latency_bucket",
		"controller_client_latency_bucket",
		"domainmapping_client_latency_bucket",
		"domainmapping_webhook_client_latency_bucket",
		"webhook_client_latency_bucket",
	}
	for _, metric := range servingMetrics {
		path := fmt.Sprintf("%s%s", pc.queryPath, metric)
		metricsURL, err := url.Parse(path)
		if err != nil {
			caCtx.T.Fatalf("Error parsing url for metrics %v", err)
		}
		// Wait until the endpoint is actually working and we get the expected value back
		_, err = pkgTest.WaitForEndpointState(
			context.Background(),
			caCtx.Clients.Kube,
			caCtx.T.Logf,
			metricsURL,
			pkgTest.EventuallyMatchesBody(metric),
			"WaitForMetricsToServeText",
			true, pc.options...)
		if err != nil {
			caCtx.T.Fatalf("Failed to access the Prometheus API endpoint and get the metric value expected: %v", err)
		}
	}
}

func getBearerTokenForPrometheusAccount(caCtx *test.Context) string {
	sa, err := caCtx.Clients.Kube.CoreV1().ServiceAccounts("openshift-monitoring").Get(context.Background(), "prometheus-k8s", meta.GetOptions{})
	if err != nil {
		caCtx.T.Fatalf("Error getting service account prometheus-k8s: %v", err)
	}
	tokenSecret := getSecretNameForToken(sa.Secrets)
	if tokenSecret == nil {
		caCtx.T.Fatal("Token name for prometheus-k8s service account not found")
	}
	sec, err := caCtx.Clients.Kube.CoreV1().Secrets("openshift-monitoring").Get(context.Background(), *tokenSecret, meta.GetOptions{})
	if err != nil {
		caCtx.T.Fatalf("Error getting secret %s: %v", *tokenSecret, err)
	}
	tokenContents := sec.Data["token"]
	if len(tokenContents) == 0 {
		caCtx.T.Fatalf("Token data is missing for token %s", *tokenSecret)
	}
	return string(tokenContents)
}

func getSecretNameForToken(secrets []corev1.ObjectReference) *string {
	for _, sec := range secrets {
		if strings.Contains(sec.Name, "token") {
			return &sec.Name
		}
	}
	return nil
}

func eventuallyMatchesValue(expectedValue string) spoof.ResponseChecker {
	return func(resp *spoof.Response) (bool, error) {
		if first, err := getFirstValueFromPromQuery(resp.Body); err != nil || first != expectedValue {
			return false, nil
		}
		return true, nil
	}
}

// Re-uses the approach in cluster-monitoring-operator test framework (https://github.com/openshift/cluster-monitoring-operator)
func getFirstValueFromPromQuery(body []byte) (string, error) {
	res, err := gabs.ParseJSON(body)
	if err != nil {
		return "", err
	}
	count, err := res.ArrayCountP("data.result")
	if err != nil {
		return "", err
	}
	if count != 1 {
		return "", fmt.Errorf("expected body to contain single timeseries but got %v", count)
	}
	timeseries, err := res.ArrayElementP(0, "data.result")
	if err != nil {
		return "", err
	}
	value, err := timeseries.ArrayElementP(1, "value")
	if err != nil {
		return "", err
	}
	return value.Data().(string), nil
}
