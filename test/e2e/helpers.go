package e2e

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/openshift-knative/serverless-operator/test"
	v1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/spoof"
)

const servingNamespace = "knative-serving"

func SetupMetricsRoute(caCtx *test.Context, name string) (*v1.Route, error) {
	routeName := "metrics-" + name
	metricsRoute := &v1.Route{
		ObjectMeta: meta.ObjectMeta{
			Name:      routeName,
			Namespace: "openshift-serverless",
		},
		Spec: v1.RouteSpec{
			Port: &v1.RoutePort{
				TargetPort: intstr.FromString("8383"),
			},
			Path: "/metrics",
			To: v1.RouteTargetReference{
				Kind: "Service",
				Name: "knative-openshift-metrics2",
			},
		},
	}
	r, err := caCtx.Clients.Route.Routes("openshift-serverless").Create(context.Background(), metricsRoute, meta.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating service monitor route: %v", err)
	}
	caCtx.AddToCleanup(func() error {
		caCtx.T.Logf("Cleaning up service metrics route: %s", routeName)
		return caCtx.Clients.Route.Routes("openshift-serverless").Delete(context.Background(), routeName, meta.DeleteOptions{})
	})
	return r, nil
}

func VerifyHealthStatusMetric(caCtx *test.Context, metricsPath string, metricLabel string, expectedValue int) {
	// Check if Operator's service monitor service is available
	_, err := caCtx.Clients.Kube.CoreV1().Services("openshift-serverless").Get(context.Background(), "knative-openshift-metrics2", meta.GetOptions{})
	if err != nil {
		caCtx.T.Fatalf("Error getting service monitor service: %v", err)
	}
	metricsURL, err := url.Parse(metricsPath)
	if err != nil {
		caCtx.T.Fatalf("Error parsing url for metrics: %v", err)
	}
	expectedStr := fmt.Sprintf(`knative_up{type="%s"} %d`, metricLabel, expectedValue)
	// Wait until the endpoint is actually working and we get the expected value back
	_, err = pkgTest.WaitForEndpointState(
		context.Background(),
		caCtx.Clients.Kube,
		caCtx.T.Logf,
		metricsURL,
		pkgTest.EventuallyMatchesBody(expectedStr),
		"WaitForMetricsToServeText",
		true)
	if err != nil {
		caCtx.T.Fatalf("Failed to access the operator metrics endpoint and get the metric value expected: %v", err)
	}
}

func setupServingControlPlaneMetricsRoutes(caCtx *test.Context, serviceMonitorServiceName string) (*v1.Route, error) {
	routeName := serviceMonitorServiceName
	metricsRoute := &v1.Route{
		ObjectMeta: meta.ObjectMeta{
			Name:      routeName,
			Namespace: servingNamespace,
		},
		Spec: v1.RouteSpec{
			Port: &v1.RoutePort{
				TargetPort: intstr.FromString("8444"),
			},
			To: v1.RouteTargetReference{
				Kind: "Service",
				Name: serviceMonitorServiceName,
			},
			TLS: &v1.TLSConfig{
				Termination: v1.TLSTerminationReencrypt,
			},
		}}
	r, err := caCtx.Clients.Route.Routes(servingNamespace).Create(context.Background(), metricsRoute, meta.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating service monitor route: %v", err)
	}
	caCtx.AddToCleanup(func() error {
		caCtx.T.Logf("Cleaning up service metrics route: %s", routeName)
		return caCtx.Clients.Route.Routes("openshift-serverless").Delete(context.Background(), routeName, meta.DeleteOptions{})
	})
	return r, nil
}

func verifyServingControlPlaneMetrics(caCtx *test.Context) {
	serviceMonitorsInstances := []struct {
		name        string
		expectedStr string
	}{{
		name:        "activator-sm",
		expectedStr: `activator_client_results{name=""}`,
	}, {
		name:        "autoscaler-sm",
		expectedStr: "autoscaler_actual_pods",
	}, {
		name:        "autoscaler-hpa-sm",
		expectedStr: "hpaautoscaler_client_latency_bucket",
	}, {
		name:        "controller-sm",
		expectedStr: "controller_client_latency_bucket",
	}, {
		name:        "domain-mapping-sm",
		expectedStr: "domainmapping_client_latency_bucket",
	}, {
		name:        "domainmapping-webhook-sm",
		expectedStr: "domainmapping_webhook_client_latency_bucket",
	}, {
		name:        "webhook-sm",
		expectedStr: "webhook_client_latency_bucket",
	}}
	for _, sm := range serviceMonitorsInstances {
		serviceName := sm.name + "-service"
		_, err := caCtx.Clients.Kube.CoreV1().Services(servingNamespace).Get(context.Background(), serviceName, meta.GetOptions{})
		if err != nil {
			caCtx.T.Fatalf("Error getting service monitor service: %v", err)
		}
		route, err := setupServingControlPlaneMetricsRoutes(caCtx, serviceName)
		if err != nil {
			caCtx.T.Fatalf("Failed to setup operator metrics route: %v", err)
		}
		metricsPath := "https://" + route.Spec.Host + "/metrics"
		metricsURL, err := url.Parse(metricsPath)
		if err != nil {
			caCtx.T.Fatalf("Error parsing url for metrics: %v", err)
		}
		bToken := getBearerTokenForAuthorizedAccount(caCtx)
		var reqOption pkgTest.RequestOption = func(request *http.Request) {
			request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bToken))
		}
		var transportOption spoof.TransportOption = func(transport *http.Transport) *http.Transport {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			return transport
		}
		// Wait until the endpoint is actually working and we get the expected value back
		_, err = pkgTest.WaitForEndpointState(
			context.Background(),
			caCtx.Clients.Kube,
			caCtx.T.Logf,
			metricsURL,
			pkgTest.EventuallyMatchesBody(sm.expectedStr),
			"WaitForMetricsToServeText",
			true, reqOption, transportOption)
		if err != nil {
			caCtx.T.Fatalf("Failed to access the operator metrics endpoint and get the metric value expected: %v", err)
		}
	}
}

func getBearerTokenForAuthorizedAccount(caCtx *test.Context) string {
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
