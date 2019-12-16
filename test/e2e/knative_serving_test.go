package e2e

import (
	"bytes"
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
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

func TestCurlKnativeMetricsUrl(t *testing.T) {

	// This test case deploys a public image (curlimages/curl) in a pod belongs
	// a namespace which is not a member of service mesh.   The expectation is
	// this this pod does not have permission to access (curl) knative controller/
	// activator/etc metrics.   This test will do just that (to curl knative metrics URL)
	// on the test pod.   The curl attempt will timeout and we assert for connection
	// timeout error.

	// Load kubeconfig
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	// Get a rest.Config from the kubeconfig file.  This will be passed into all
	// the client objects we create.
	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		t.Fatal("Failed", err)
	}

	// "default" namespace must not be a member of SMMR!
	const namespace = "default"

	// Create a Kubernetes core/v1 client.
	coreclient, err := corev1client.NewForConfig(restconfig)
	if err != nil {
		t.Fatal("Failed", err)
	}

	// Create a busybox Pod.  By running `cat`, the Pod will sit and do nothing.
	var zero int64
	pod, err := coreclient.Pods(namespace).Create(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "curlbox",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "curl-container",
					Image:   "curlimages/curl",
					Command: []string{"cat"},
					Stdin:   true,
				},
			},
			TerminationGracePeriodSeconds: &zero,
		},
	})
	if err != nil {
		t.Fatal("Failed", err)
	}

	// Delete the Pod before we exit.
	defer coreclient.Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})

	// Wait for the Pod to indicate Ready == True.
	watcher, err := coreclient.Pods(namespace).Watch(
		metav1.SingleObject(pod.ObjectMeta),
	)
	if err != nil {
		t.Fatal("Failed", err)
	}

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Modified:
			pod = event.Object.(*corev1.Pod)

			// If the Pod contains a status condition Ready == True, stop
			// watching.
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady &&
					cond.Status == corev1.ConditionTrue {
					watcher.Stop()
				}
			}

		default:
			t.Fatal("unexpected event type " + event.Type)
		}
	}

	// turn off stdin and termial i/o
	const tty = false

	req := coreclient.RESTClient().Post().
		Resource("pods").
		Name(pod.GetName()).
		Namespace(pod.GetNamespace()).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   []string{"curl", "controller.knative-serving.svc:9090/metrics"},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       tty,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restconfig, "POST", req.URL())
	if err != nil {
		t.Fatal("Failed", err)
	}

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: outBuf,
		Stderr: errBuf,
		Tty:    tty,
	})

	if err == nil {
		t.Fatal("Curl command is expected to fail but it did not.")
	}

	errMessage := errBuf.String()
	if !strings.Contains(errMessage, "Failed to connect to controller.knative-serving.svc port 9090: Operation timed out") {
		t.Fatalf("Unexpected error: %s", errMessage)
	}

}

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

	t.Run("deploy knative and kubernetes service in same namespace", func(t *testing.T) {
		testKnativeVersusKubeServicesInOneNamespace(t, caCtx)
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

	t.Run("undeploy serverless operator and check dependent operators removed", func(t *testing.T) {
		caCtx.Cleanup()
		err := test.WaitForOperatorDepsDeleted(caCtx)
		if err != nil {
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
	ksvc, err = test.WithServiceReady(caCtx, helloworldService, testNamespace2, image)
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
