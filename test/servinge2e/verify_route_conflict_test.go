package servinge2e

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// This test creates two services: conflict-test/a and test/a-conflict. Due to the domain template,
// those two will clash in the generated URL. The test then verifies that the "older" service "wins".
func TestRouteConflictBehavior(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	svcA := types.NamespacedName{Namespace: "conflict-test", Name: "a"}
	svcB := types.NamespacedName{Namespace: "test", Name: "a-conflict"}

	for _, ns := range []string{svcA.Namespace, svcB.Namespace} {
		if _, err := caCtx.Clients.Kube.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}, metav1.CreateOptions{}); err != nil && !apierrs.IsAlreadyExists(err) {
			t.Fatalf("Failed to create namespace %s: %v", ns, err)
		}
		defer caCtx.Clients.Kube.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
	}

	// Each of the two services should be the "oldest" and it should work regardless.
	for _, services := range [][]types.NamespacedName{{svcA, svcB}, {svcB, svcA}} {
		older := services[0]
		newer := services[1]

		t.Logf("older: %v, newer: %v", older, newer)

		olderSvc, err := test.WithServiceReady(caCtx, older.Name, older.Namespace, image)
		if err != nil {
			t.Fatal("Knative Service not ready", err)
		}

		WaitForRouteServingText(t, caCtx, olderSvc.Status.URL.URL(), helloworldText)

		_, err = test.CreateService(caCtx, newer.Name, newer.Namespace, image)
		if err != nil {
			t.Fatal("Failed to create conflicting Knative Service", err)
		}

		if _, err := test.WaitForServiceState(caCtx, newer.Name, newer.Namespace, func(s *servingv1.Service, err error) (bool, error) {
			if err != nil {
				return false, err
			}

			// Wait until a revision is ready.
			if s.Status.LatestReadyRevisionName == "" {
				return false, nil
			}

			for _, cond := range s.Status.Conditions {
				// Wait until we see "DomainConflict"
				if cond.Reason == "DomainConflict" {
					return true, nil
				}
			}
			return false, nil
		}); err != nil {
			t.Fatal("Desired state never occurred", err)
		}

		// Verify that the "older" service still works.
		resp, err := http.Get(olderSvc.Status.URL.String())
		if err != nil {
			t.Fatalf("Failed sending request to %s: %v", olderSvc.Status.URL.String(), err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal("Failed to read body", err)
		}
		bodyStr := strings.TrimSpace(string(body))

		if resp.StatusCode != http.StatusOK || bodyStr != helloworldText {
			t.Fatalf("Received wrong response, status %v, body %v", resp.StatusCode, bodyStr)
		}

		for _, svc := range services {
			if err := caCtx.Clients.Serving.ServingV1().Services(svc.Namespace).Delete(context.Background(), svc.Name, metav1.DeleteOptions{}); err != nil {
				t.Fatalf("Failed to remove ksvc %v: %v", svc, err)
			}
		}
	}
}
