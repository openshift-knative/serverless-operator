package e2e

import (
	"strings"
	"testing"

	servingv1beta1 "github.com/knative/serving/pkg/apis/serving/v1beta1"
	"github.com/openshift-knative/serverless-operator/test"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgTest "knative.dev/pkg/test"
	servingoperatorv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

const (
	knativeServing                = "knative-serving"
	testNamespace                 = "serverless-tests"
	testNamespace2                = "serverless-tests2"
	image                         = "gcr.io/knative-samples/helloworld-go"
	proxyImage                    = "gcr.io/knative-samples/autoscale-go:0.1"
	helloworldService             = "helloworld-go"
	helloworldService2            = "helloworld-go2"
	kubeHelloworldService         = "kube-helloworld-go"
	helloworldText                = "Hello World!"
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
		if _, err := v1a1test.WithKnativeServingReady(caCtx, knativeServing, knativeServing); err != nil {
			t.Fatal("Failed to deploy KnativeServing", err)
		}
	})

	t.Run("verify correct deployment shape", func(t *testing.T) {
		api, err := caCtx.Clients.KubeAggregator.ApiregistrationV1beta1().APIServices().Get("v1beta1.custom.metrics.k8s.io", metav1.GetOptions{})
		// We're good if no APIService exists at all
		if err != nil && !apierrs.IsNotFound(err) {
			t.Fatalf("Failed to fetch APIService: %v", err)
		}

		if api != nil && api.Spec.Service != nil && api.Spec.Service.Namespace == knativeServing && api.Spec.Service.Name == "autoscaler" {
			t.Fatalf("Found a custom-metrics API registered at the autoscaler")
		}

		// The list of deployments that are HA-ed
		for _, deployment := range []string{"controller", "autoscaler-hpa"} {
			if err := test.CheckDeploymentScale(caCtx, knativeServing, deployment, haReplicas); err != nil {
				t.Fatalf("Failed to verify default HA settings: %v", err)
			}
		}
	})

	t.Run("deploy knative service using kubeadmin", func(t *testing.T) {
		if _, err := test.WithServiceReady(caCtx, helloworldService, testNamespace, image); err != nil {
			t.Fatal("Knative Service not ready", err)
		}
	})

	t.Run("user permissions", func(t *testing.T) {
		testUserPermissions(t)
	})

	t.Run("deploy knative and kubernetes service in same namespace", func(t *testing.T) {
		testKnativeVersusKubeServicesInOneNamespace(t, caCtx)
	})

	t.Run("update global proxy and verify calls goes through proxy server", func(t *testing.T) {
		t.Skip("SRKVS-462: This test needs thorough hardening")
		testKnativeServingForGlobalProxy(t, caCtx)
	})

	t.Run("remove knativeserving cr", func(t *testing.T) {
		if err := v1a1test.DeleteKnativeServing(caCtx, knativeServing, knativeServing); err != nil {
			t.Fatal("Failed to remove Knative Serving", err)
		}

		ns, err := caCtx.Clients.Kube.CoreV1().Namespaces().Get(knativeServing+"-ingress", metav1.GetOptions{})
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

func testKnativeVersusKubeServicesInOneNamespace(t *testing.T, caCtx *test.Context) {
	//Create deployment
	err := test.CreateDeployment(caCtx, kubeHelloworldService, testNamespace2, image)
	if err != nil {
		t.Fatal("Deployment not created", err)
	}
	// Deploy plain Kube service
	svc, err := test.CreateKubeService(caCtx, kubeHelloworldService, testNamespace2)
	if err != nil {
		t.Fatal("Kubernetes service not created", err)
	}
	route, err := test.WithRouteForServiceReady(caCtx, svc.Name, testNamespace2)
	if err != nil {
		t.Fatal("Failed to create route for service", svc.Name, err)
	}
	kubeServiceURL := "http://" + route.Status.Ingress[0].Host

	// Check Kube service responds
	waitForRouteServingText(t, caCtx, kubeServiceURL, helloworldText)

	// Deploy Knative service in the same namespace
	ksvc, err := test.WithServiceReady(caCtx, helloworldService, testNamespace2, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Check that both services respond
	waitForRouteServingText(t, caCtx, ksvc.Status.URL.Host, helloworldText)
	waitForRouteServingText(t, caCtx, kubeServiceURL, helloworldText)

	// Delete Knative service
	caCtx.Clients.Serving.ServingV1beta1().Services(testNamespace2).Delete(ksvc.Name, &metav1.DeleteOptions{})

	// Check that Kube service still responds
	waitForRouteServingText(t, caCtx, kubeServiceURL, helloworldText)

	// Remove the Kube service
	caCtx.Clients.Route.Routes(testNamespace2).Delete(svc.Name, &metav1.DeleteOptions{})
	caCtx.Clients.Kube.CoreV1().Services(testNamespace2).Delete(svc.Name, &metav1.DeleteOptions{})
	caCtx.Clients.Kube.AppsV1().Deployments(testNamespace2).Delete(svc.Name, &metav1.DeleteOptions{})

	// Deploy Knative service in the namespace first
	ksvc, err = test.WithServiceReady(caCtx, helloworldService2, testNamespace2, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Check that Knative service responds
	waitForRouteServingText(t, caCtx, ksvc.Status.URL.Host, helloworldText)

	//Create deployment
	err = test.CreateDeployment(caCtx, kubeHelloworldService, testNamespace2, image)
	if err != nil {
		t.Fatal("Deployment not created", err)
	}
	// Deploy plain Kube service
	svc, err = test.CreateKubeService(caCtx, kubeHelloworldService, testNamespace2)
	if err != nil {
		t.Fatal("Kubernetes service not created", err)
	}
	route, err = test.WithRouteForServiceReady(caCtx, svc.Name, testNamespace2)
	if err != nil {
		t.Fatal("Failed to create route for service", svc.Name, err)
	}
	kubeServiceURL = "http://" + route.Status.Ingress[0].Host

	// Check that both services respond
	waitForRouteServingText(t, caCtx, ksvc.Status.URL.Host, helloworldText)
	waitForRouteServingText(t, caCtx, kubeServiceURL, helloworldText)

	// Remove the Kube service
	caCtx.Clients.Route.Routes(testNamespace2).Delete(svc.Name, &metav1.DeleteOptions{})
	caCtx.Clients.Kube.CoreV1().Services(testNamespace2).Delete(svc.Name, &metav1.DeleteOptions{})
	caCtx.Clients.Kube.AppsV1().Deployments(testNamespace2).Delete(svc.Name, &metav1.DeleteOptions{})

	// Check that Knative service still responds
	waitForRouteServingText(t, caCtx, ksvc.Status.URL.Host, helloworldText)

	// Delete the Knative service
	caCtx.Clients.Serving.ServingV1beta1().Services(testNamespace2).Delete(ksvc.Name, &metav1.DeleteOptions{})
}

func testUserPermissions(t *testing.T) {
	paCtx := test.SetupProjectAdmin(t)
	editCtx := test.SetupEdit(t)
	viewCtx := test.SetupView(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, paCtx, editCtx, viewCtx) })
	defer test.CleanupAll(t, paCtx, editCtx, viewCtx)

	tests := []struct {
		name        string
		userContext *test.Context
		operation   func(context *test.Context) error
		wantErrStr  string
	}{{
		name: "user with view role can get",
		operation: func(c *test.Context) error {
			_, err := c.Clients.Serving.ServingV1beta1().Services(testNamespace).Get(helloworldService, metav1.GetOptions{})
			return err
		},
		userContext: viewCtx,
	}, {
		name: "user with view role can list",
		operation: func(c *test.Context) error {
			_, err := c.Clients.Serving.ServingV1beta1().Services(testNamespace).List(metav1.ListOptions{})
			return err
		},
		userContext: viewCtx,
	}, {
		name: "user with view role cannot create",
		operation: func(c *test.Context) error {
			_, err := test.CreateService(c, "userview-service", testNamespace, image)
			return err
		},
		userContext: viewCtx,
		wantErrStr:  "is forbidden",
	}, {
		name: "user with view role cannot delete",
		operation: func(c *test.Context) error {
			return c.Clients.Serving.ServingV1beta1().Services(testNamespace).Delete(helloworldService, &metav1.DeleteOptions{})
		},
		userContext: viewCtx,
		wantErrStr:  "is forbidden",
	}, {
		name: "user with project admin role can get",
		operation: func(c *test.Context) error {
			_, err := c.Clients.Serving.ServingV1beta1().Services(testNamespace).Get(helloworldService, metav1.GetOptions{})
			return err
		},
		userContext: paCtx,
	}, {
		name: "user with project admin role can list",
		operation: func(c *test.Context) error {
			_, err := c.Clients.Serving.ServingV1beta1().Services(testNamespace).List(metav1.ListOptions{})
			return err
		},
		userContext: paCtx,
	}, {
		name: "user with project admin role can create",
		operation: func(c *test.Context) error {
			_, err := test.CreateService(c, "projectadmin-service", testNamespace, image)
			return err
		},
		userContext: paCtx,
	}, {
		name: "user with project admin role can delete",
		operation: func(c *test.Context) error {
			return c.Clients.Serving.ServingV1beta1().Services(testNamespace).Delete("projectadmin-service", &metav1.DeleteOptions{})
		},
		userContext: paCtx,
	}, {
		name: "user with edit role can get",
		operation: func(c *test.Context) error {
			_, err := c.Clients.Serving.ServingV1beta1().Services(testNamespace).Get(helloworldService, metav1.GetOptions{})
			return err
		},
		userContext: editCtx,
	}, {
		name: "user with edit role can list",
		operation: func(c *test.Context) error {
			_, err := c.Clients.Serving.ServingV1beta1().Services(testNamespace).List(metav1.ListOptions{})
			return err
		},
		userContext: editCtx,
	}, {
		name: "user with edit role can create",
		operation: func(c *test.Context) error {
			_, err := test.CreateService(c, "useredit-service", testNamespace, image)
			return err
		},
		userContext: editCtx,
	}, {
		name: "user with edit role can delete",
		operation: func(c *test.Context) error {
			return c.Clients.Serving.ServingV1beta1().Services(testNamespace).Delete("useredit-service", &metav1.DeleteOptions{})

		},
		userContext: editCtx,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.operation(test.userContext)
			if (err != nil) != (test.wantErrStr != "") {
				t.Errorf("User with role %s has unexpected behavior on knative services. Error thrown: %v, error expected: %t", test.userContext.Name, err, (test.wantErrStr != ""))
			}
			if err != nil && !strings.Contains(err.Error(), test.wantErrStr) {
				t.Errorf("Unexpected error for user with role %s: %v", test.userContext.Name, err)
			}
		})
	}
}

func waitForRouteServingText(t *testing.T, caCtx *test.Context, routeDomain, expectedText string) {
	t.Helper()
	if _, err := pkgTest.WaitForEndpointState(
		&pkgTest.KubeClient{Kube: caCtx.Clients.Kube},
		t.Logf,
		routeDomain,
		pkgTest.EventuallyMatchesBody(expectedText),
		"WaitForRouteToServeText",
		true); err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected text \"%s\": %v", routeDomain, expectedText, err)
	}
}

func testKnativeServingForGlobalProxy(t *testing.T, caCtx *test.Context) {
	cleanup := func() {
		if err := test.UpdateGlobalProxy(caCtx, ""); err != nil {
			t.Fatal("Failed to update proxy", err)
		}
		// In order to make sure state of the knative serving same like before
		if _, err := v1a1test.WaitForKnativeServingState(caCtx, knativeServing, knativeServing, func(ks *servingoperatorv1alpha1.KnativeServing, err error) (bool, error) {
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
	if err := test.WaitForControllerEnvironment(caCtx, knativeServing, httpProxy, "http://"+proxyIP); err != nil {
		t.Fatal(err)
	}

	t.Log("deploy knative service after proxy update")
	if _, err := test.CreateService(caCtx, proxyHelloworldService, testNamespace, proxyImage); err != nil {
		t.Fatal("Failed to create service", err)
	}
	svcState, err := test.WaitForServiceState(caCtx, proxyHelloworldService, testNamespace, func(s *servingv1beta1.Service, err error) (bool, error) {
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
