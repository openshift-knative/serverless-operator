package servinge2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	networkingv1 "k8s.io/api/networking/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	policyNameDeny  = "deny-all"
	policyNameAllow = "allow-from-serving-system-ns"
)

// This test creates two networkpolicies.
// 1. creates the deny-all policy and verify if access does not work.
// 2. create the allow-from-serving-system-ns and verify if access works.
func TestNetworkPolicy(t *testing.T) {
	t.Skip("TODO: figure out why this won't work on a local CRC")
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	ksvc, err := test.WithServiceReady(caCtx, "networkpolicy-test", testNamespace3, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: policyNameDeny,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			Ingress:     []networkingv1.NetworkPolicyIngressRule{},
		},
	}

	_, err = caCtx.Clients.Kube.NetworkingV1().NetworkPolicies(testNamespace3).Create(policy)
	if err != nil && !apierrs.IsAlreadyExists(err) {
		t.Fatalf("Failed to create networkpolicy %v: %v", policy, err)
	}
	defer caCtx.Clients.Kube.NetworkingV1().NetworkPolicies(testNamespace3).Delete(policyNameDeny, &metav1.DeleteOptions{})

	// Poll until network policy became active. It takes a few seconds.
	err = wait.PollImmediate(test.Interval, 1*time.Minute, func() (bool, error) {
		_, inErr := http.Get(ksvc.Status.URL.String())
		if inErr == nil {
			t.Logf("Network policy did not block the request to %s", ksvc.Status.URL.String())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Network policy did not block the request: %v", err)
	}

	policyAllow := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: policyNameAllow,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"serving.knative.openshift.io/system-namespace": "true",
						},
					},
				}},
			}},
		},
	}

	_, err = caCtx.Clients.Kube.NetworkingV1().NetworkPolicies(testNamespace3).Create(policyAllow)
	if err != nil && !apierrs.IsAlreadyExists(err) {
		t.Fatalf("Failed to create networkpolicy %v: %v", policyAllow, err)
	}
	defer caCtx.Clients.Kube.NetworkingV1().NetworkPolicies(testNamespace3).Delete(policyNameAllow, &metav1.DeleteOptions{})

	// Poll until network policy became active. It takes a few seconds.
	err = wait.PollImmediate(test.Interval, 1*time.Minute, func() (bool, error) {
		_, inErr := http.Get(ksvc.Status.URL.String())
		if inErr != nil {
			t.Logf("Network policy did not allow the request to %s: %v", ksvc.Status.URL.String(), inErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Network policy did not allow the request: %v", err)
	}
}
