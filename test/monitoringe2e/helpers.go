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

	"github.com/openshift-knative/serverless-operator/test"
	v1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	prom "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
)

var prometheusTargetTimeout = 20 * time.Minute

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

func getPrometheusRoute(caCtx *test.Context) (*v1.Route, error) {
	r, err := caCtx.Clients.Route.Routes("openshift-monitoring").Get(context.Background(), "prometheus-k8s", meta.GetOptions{})
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
		value, _, err := pc.Query(context.Background(), url.QueryEscape(fmt.Sprintf(`knative_up{type="%s"}`, label)), time.Time{})
		if err != nil {
			return false, err
		}
		return value.String() == expectedValue, nil
	}); err != nil {
		return fmt.Errorf("failed to access the Prometheus API endpoint and get the metric value expected: %w", err)
	}
	return nil
}

func VerifyServingControlPlaneMetrics(caCtx *test.Context) error {
	pc, err := newPrometheusClient(caCtx)
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
		if err := wait.PollImmediate(test.Interval, prometheusTargetTimeout, func() (bool, error) {
			value, _, err := pc.Query(context.Background(), metric, time.Time{})
			if err != nil {
				return false, err
			}
			return value.Type() == prommodel.ValScalar, nil
		}); err != nil {
			return fmt.Errorf("failed to access the Prometheus API endpoint for %s and get the metric value expected: %w", metric, err)
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
