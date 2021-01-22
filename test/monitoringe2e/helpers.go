package monitoringe2e

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/openshift-knative/serverless-operator/test"
	v1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/spoof"
)

var prometheusTargetTimeout = 20 * time.Minute

type prometheusInfo struct {
	options   []interface{}
	queryPath string
	token     string
}

func newPrometheusInfo(caCtx *test.Context) (*prometheusInfo, error) {
	route, err := getPrometheusRoute(caCtx)
	if err != nil {
		return nil, err
	}
	bToken, err := getBearerTokenForPrometheusAccount(caCtx)
	if err != nil {
		return nil, err
	}
	var reqOption pkgTest.RequestOption = func(request *http.Request) {
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bToken))
	}
	var transportOption spoof.TransportOption = func(transport *http.Transport) *http.Transport {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return transport
	}
	pc := &prometheusInfo{
		options:   []interface{}{reqOption, transportOption},
		queryPath: "https://" + route.Spec.Host + "/api/v1/query?query=",
		token:     bToken,
	}
	return pc, nil
}

func getPrometheusRoute(caCtx *test.Context) (*v1.Route, error) {
	r, err := caCtx.Clients.Route.Routes("openshift-monitoring").Get(context.Background(), "prometheus-k8s", meta.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting Prometheus route %v", err)
	}
	return r, nil
}

func VerifyHealthStatusMetric(caCtx *test.Context, label string, expectedValue string) error {
	pc, err := newPrometheusInfo(caCtx)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("%s%s", pc.queryPath, url.QueryEscape(fmt.Sprintf(`knative_up{type="%s"}`, label)))
	metricsURL, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("error parsing url for metrics: %v", err)
	}
	// Wait until the endpoint is actually working and we get the expected value back
	_, err = pkgTest.WaitForEndpointStateWithTimeout(
		context.Background(),
		caCtx.Clients.Kube,
		caCtx.T.Logf,
		metricsURL,
		eventuallyMatchesValue(expectedValue),
		"WaitForMetricsToServeText",
		true,
		prometheusTargetTimeout, pc.options...)
	if err != nil {
		return fmt.Errorf("failed to access the Prometheus API endpoint and get the metric value expected: %v", err)
	}
	return nil
}

func VerifyServingControlPlaneMetrics(caCtx *test.Context) error {
	pc, err := newPrometheusInfo(caCtx)
	if err != nil {
		return err
	}
	servingMetrics := []string{
		"activator_go_mallocs",
		"autoscaler_go_mallocs",
		"hpaautoscaler_go_mallocs",
		"controller_go_mallocs",
		"domainmapping_go_mallocs",
		"domainmapping_webhook_go_mallocs",
		"webhook_go_mallocs",
	}
	for _, metric := range servingMetrics {
		path := fmt.Sprintf("%s%s", pc.queryPath, metric)
		metricsURL, err := url.Parse(path)
		if err != nil {
			return fmt.Errorf("error parsing url for metrics %v", err)
		}
		// Wait until the endpoint is actually working and we get the expected value back
		_, err = pkgTest.WaitForEndpointStateWithTimeout(
			context.Background(),
			caCtx.Clients.Kube,
			caCtx.T.Logf,
			metricsURL,
			pkgTest.EventuallyMatchesBody(metric),
			"WaitForMetricsToServeText",
			true,
			prometheusTargetTimeout, pc.options...)
		if err != nil {
			return fmt.Errorf("failed to access the Prometheus API endpoint for %s and get the metric value expected: %v", metric, err)
		}
	}
	return nil
}

func getBearerTokenForPrometheusAccount(caCtx *test.Context) (string, error) {
	sa, err := caCtx.Clients.Kube.CoreV1().ServiceAccounts("openshift-monitoring").Get(context.Background(), "prometheus-k8s", meta.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting service account prometheus-k8s %v", err)
	}
	tokenSecret := getSecretNameForToken(sa.Secrets)
	if tokenSecret == "" {
		return "", errors.New("token name for prometheus-k8s service account not found")
	}
	sec, err := caCtx.Clients.Kube.CoreV1().Secrets("openshift-monitoring").Get(context.Background(), tokenSecret, meta.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting secret %s %v", tokenSecret, err)
	}
	tokenContents := sec.Data["token"]
	if len(tokenContents) == 0 {
		return "", fmt.Errorf("token data is missing for token %s", tokenSecret)
	}
	return string(tokenContents), nil
}

func getSecretNameForToken(secrets []corev1.ObjectReference) string {
	for _, sec := range secrets {
		if strings.Contains(sec.Name, "token") {
			return sec.Name
		}
	}
	return ""
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
