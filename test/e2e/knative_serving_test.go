package e2e

import (
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgTest "knative.dev/pkg/test"
)

const (
	knativeServing        = "knative-serving"
	testNamespace         = "serverless-tests"
	testNamespace2        = "serverless-tests2"
	image                 = "gcr.io/knative-samples/helloworld-go"
	helloworldService     = "helloworld-go"
	kubeHelloworldService = "kube-helloworld-go"
	helloworldText        = "Hello World!"
)

func TestKnativeServing(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	paCtx := test.SetupProjectAdmin(t)
	editCtx := test.SetupEdit(t)
	viewCtx := test.SetupView(t)

	defer test.CleanupAll(caCtx, paCtx, editCtx, viewCtx)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(caCtx, paCtx, editCtx, viewCtx) })

	t.Run("create subscription and wait for CSV to succeed", func(t *testing.T) {
		_, err := test.WithOperatorReady(caCtx, "serverless-operator-subscription")
		if err != nil {
			t.Fatal("Failed", err)
		}
	})

	t.Run("deploy knativeserving cr and wait for it to be ready", func(t *testing.T) {
		_, err := test.WithKnativeServingReady(caCtx, knativeServing, knativeServing)
		if err != nil {
			t.Fatal("Failed to deploy KnativeServing", err)
		}
	})

	t.Run("deploy knative service using kubeadmin", func(t *testing.T) {
		_, err := test.WithServiceReady(caCtx, helloworldService, testNamespace, image)
		if err != nil {
			t.Fatal("Knative Service not ready", err)
		}
	})

	t.Run("user permissions", func(t *testing.T) {
		testUserPermissions(t, paCtx, editCtx, viewCtx)
	})

	t.Run("remove knativeserving cr", func(t *testing.T) {
		if err := test.DeleteKnativeServing(caCtx, knativeServing, knativeServing); err != nil {
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

	t.Run("deploy knative and kubernetes service in same namespace", func(t *testing.T) {
		testKnativeVersusKubeServicesInOneNamespace(t, caCtx)
	})

	t.Run("undeploy serverless operator and check dependent operators removed", func(t *testing.T) {
		caCtx.Cleanup()
		err := test.WaitForOperatorDepsDeleted(caCtx)
		if err != nil {
			t.Fatalf("Operators still running: %v", err)
		}
	})
}

func testKnativeVersusKubeServicesInOneNamespace(t *testing.T, caCtx *test.Context) {
	// Deploy plain Kube service
	svc, err := test.CreateKubeService(caCtx, kubeHelloworldService, testNamespace2, image)
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
	test.DeleteService(caCtx, ksvc.Name, testNamespace2)

	// Check that Kube service still responds
	waitForRouteServingText(t, caCtx, kubeServiceURL, helloworldText)

	// Remove the Kube service
	test.DeleteRoute(caCtx, svc.Name, testNamespace2)
	test.DeleteKubeService(caCtx, svc.Name, testNamespace2)
	test.DeleteDeployment(caCtx, svc.Name, testNamespace2)

	// Deploy Knative service in the namespace first
	ksvc, err = test.WithServiceReady(caCtx, helloworldService, testNamespace2, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Check that Knative service responds
	waitForRouteServingText(t, caCtx, ksvc.Status.URL.Host, helloworldText)

	// Deploy plain Kube service
	svc, err = test.CreateKubeService(caCtx, kubeHelloworldService, testNamespace2, image)
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
	test.DeleteRoute(caCtx, svc.Name, testNamespace2)
	test.DeleteKubeService(caCtx, svc.Name, testNamespace2)
	test.DeleteDeployment(caCtx, svc.Name, testNamespace2)

	// Check that Knative service still responds
	waitForRouteServingText(t, caCtx, ksvc.Status.URL.Host, helloworldText)
}

func testUserPermissions(t *testing.T, paCtx *test.Context, editCtx *test.Context, viewCtx *test.Context) {
	tests := []struct {
		name        string
		userContext *test.Context
		operation   func(context *test.Context) error
		wantErrStr  string
	}{{
		name: "user with view role can get",
		operation: func(c *test.Context) error {
			_, err := test.GetService(c, helloworldService, testNamespace)
			return err
		},
		userContext: viewCtx,
	}, {
		name: "user with view role can list",
		operation: func(c *test.Context) error {
			_, err := test.ListServices(c, testNamespace)
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
			return test.DeleteService(c, helloworldService, testNamespace)
		},
		userContext: viewCtx,
		wantErrStr:  "is forbidden",
	}, {
		name: "user with project admin role can get",
		operation: func(c *test.Context) error {
			_, err := test.GetService(c, helloworldService, testNamespace)
			return err
		},
		userContext: paCtx,
	}, {
		name: "user with project admin role can list",
		operation: func(c *test.Context) error {
			_, err := test.ListServices(c, testNamespace)
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
			return test.DeleteService(c, "projectadmin-service", testNamespace)
		},
		userContext: paCtx,
	}, {
		name: "user with edit role can get",
		operation: func(c *test.Context) error {
			_, err := test.GetService(c, helloworldService, testNamespace)
			return err
		},
		userContext: editCtx,
	}, {
		name: "user with edit role can list",
		operation: func(c *test.Context) error {
			_, err := test.ListServices(c, testNamespace)
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
			return test.DeleteService(c, "useredit-service", testNamespace)
		},
		userContext: editCtx,
	},
	}

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
	_, err := pkgTest.WaitForEndpointState(
		&pkgTest.KubeClient{Kube: caCtx.Clients.Kube},
		t.Logf,
		routeDomain,
		pkgTest.EventuallyMatchesBody(expectedText),
		"WaitForRouteToServeText",
		true)
	if err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected text \"%s\": %v", routeDomain, expectedText, err)
	}
}
