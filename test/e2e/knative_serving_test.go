package e2e

import (
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingoperatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	serviceName                   = "knative-openshift-metrics"
	serviceMonitorName            = serviceName
	servingName                   = "knative-serving"
	servingNamespace              = "knative-serving"
	testNamespace                 = "serverless-tests"
	image                         = "gcr.io/knative-samples/helloworld-go"
	proxyImage                    = "gcr.io/knative-samples/autoscale-go:0.1"
	proxyHelloworldServiceSuccess = "proxy-helloworld-go-success"
	proxyHelloworldService        = "proxy-helloworld-go"
	httpProxy                     = "HTTP_PROXY"
	proxyIP                       = "1.2.4.5:8999"
	haReplicas                    = 2
)

func TestKnativeServing(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)

	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	t.Run("create subscription and wait for CSV to succeed", func(t *testing.T) {
		if _, err := test.WithOperatorReady(caCtx, "serverless-operator-subscription"); err != nil {
			t.Fatal("Failed", err)
		}
	})

	t.Run("deploy knativeserving cr and wait for it to be ready", func(t *testing.T) {
		if _, err := v1a1test.WithKnativeServingReady(caCtx, servingName, servingNamespace); err != nil {
			t.Fatal("Failed to deploy KnativeServing", err)
		}
	})

	t.Run("verify correct deployment shape", func(t *testing.T) {
		// Check the status of scaled deployments in the knative serving namespace
		for _, deployment := range []string{"activator", "controller", "autoscaler-hpa"} {
			if err := test.CheckDeploymentScale(caCtx, servingNamespace, deployment, haReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings: %v", err)
			}
		}
		// Check the status of deployments in the knative serving namespace
		for _, deployment := range []string{"autoscaler", "webhook"} {
			if _, err := test.WithDeploymentReady(caCtx, deployment, servingNamespace); err != nil {
				t.Fatalf("Deployment %s is not ready: %v", deployment, err)
			}
		}
		// Check the status of deployments in the ingress namespace.
		for _, deployment := range []string{"3scale-kourier-control", "3scale-kourier-gateway"} {
			// Workaround for https://issues.redhat.com/browse/SRVCOM-1008 - wait for Kourier deployments to
			// be ready before checking their scales.
			if _, err := test.WithDeploymentReady(caCtx, deployment, servingNamespace+"-ingress"); err != nil {
				t.Fatal("Failed", err)
			}
			if err := test.CheckDeploymentScale(caCtx, servingNamespace+"-ingress", deployment, haReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings: %v", err)
			}
		}

	})

	t.Run("make sure no gcr.io references are there", func(t *testing.T) {
		VerifyNoDisallowedImageReference(t, caCtx, servingNamespace)
	})

	t.Run("update global proxy and verify calls goes through proxy server", func(t *testing.T) {
		t.Skip("SRKVS-462: This test needs thorough hardening")
		testKnativeServingForGlobalProxy(t, caCtx)
	})

	t.Run("remove knativeserving cr", func(t *testing.T) {
		if err := v1a1test.DeleteKnativeServing(caCtx, servingName, servingNamespace); err != nil {
			t.Fatal("Failed to remove Knative Serving", err)
		}

		ns, err := caCtx.Clients.Kube.CoreV1().Namespaces().Get(servingNamespace+"-ingress", metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			// Namespace is already gone, all good!
			return
		} else if err != nil {
			t.Fatal("Failed fetching ingress namespace", err)
		}

		// If the namespace is not gone yet, check if it's terminating.
		if ns.Status.Phase != corev1.NamespaceTerminating {
			t.Fatalf("Ingress namespace phase = %v, want %v", ns.Status.Phase, corev1.NamespaceTerminating)
		}
	})

	t.Run("undeploy serverless operator and check dependent operators removed", func(t *testing.T) {
		caCtx.Cleanup(t)
		if err := test.WaitForOperatorDepsDeleted(caCtx); err != nil {
			t.Fatalf("Operators still running: %v", err)
		}
	})
}

func testKnativeServingForGlobalProxy(t *testing.T, caCtx *test.Context) {
	cleanup := func() {
		if err := test.UpdateGlobalProxy(caCtx, ""); err != nil {
			t.Fatal("Failed to update proxy", err)
		}
		// In order to make sure state of the knative serving same like before
		if _, err := v1a1test.WaitForKnativeServingState(caCtx, servingName, servingNamespace, func(ks *servingoperatorv1alpha1.KnativeServing, err error) (bool, error) {
			if apierrs.IsUnauthorized(err) {
				// Retry unauthorized errors, they sometimes happen when resetting the proxy.
				return false, nil
			}
			return v1a1test.IsKnativeServingReady(ks, err)
		}); err != nil {
			t.Fatal("knative serving is not in desired state", err)
		}
	}

	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	t.Log("update global proxy with empty proxy value")
	if err := test.UpdateGlobalProxy(caCtx, ""); err != nil {
		t.Fatal("Failed to update proxy", err)
	}

	t.Log("deploy successfully knative service after proxy update")
	if _, err := test.WithServiceReady(caCtx, proxyHelloworldServiceSuccess, testNamespace, image); err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	t.Log("update global proxy with proxy server")
	if err := test.UpdateGlobalProxy(caCtx, "http://"+proxyIP); err != nil {
		t.Fatal("Failed to update proxy", err)
	}

	t.Log("wait for controller to be ready after update")
	if err := test.WaitForControllerEnvironment(caCtx, servingNamespace, httpProxy, "http://"+proxyIP); err != nil {
		t.Fatal(err)
	}

	t.Log("deploy knative service after proxy update")
	if _, err := test.CreateService(caCtx, proxyHelloworldService, testNamespace, proxyImage); err != nil {
		t.Fatal("Failed to create service", err)
	}
	svcState, err := test.WaitForServiceState(caCtx, proxyHelloworldService, testNamespace, func(s *servingv1.Service, err error) (bool, error) {
		if err != nil {
			return false, err
		}
		for _, cond := range s.Status.Conditions {
			// After global proxy update every call goes through proxy server
			// Here it give unable to pull image because it tries to connect to not running http server
			if strings.Contains(cond.Message, "failed to fetch image information") && strings.Contains(cond.Message, proxyIP) {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		t.Fatal("service state never appeared", svcState)
	}

	// Ref: https://bugzilla.redhat.com/show_bug.cgi?id=1751903#c11
	// Currently when we update cluster proxy by removing httpProxy, noProxy etc... OLM will not update controller
	// once bugzilla issue https://bugzilla.redhat.com/show_bug.cgi?id=1751903#c11 fixes need to add test case related to
	// verifying success of proxy update and successfully deploying knative service
}
