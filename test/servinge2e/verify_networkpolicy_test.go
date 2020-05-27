package servinge2e

import (
	"net/http"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	networkingv1 "k8s.io/api/networking/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	policyNameDeny  = "deny-all"
	policyNameAllow = "allow-from-serving-system-ns"
)

// This test creates two networkpolicies.
// 1. creates the deny-all policy and verify if access does not work.
// 2. create the allow-from-serving-system-ns and verify if access works.
func TestNetworkPolicy(t *testing.T) {
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

	_, err = http.Get(ksvc.Status.URL.String())
	if err == nil {
		t.Fatalf("Netowrk policy did not block the request to %s", ksvc.Status.URL.String())
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
	_, err = http.Get(ksvc.Status.URL.String())
	if err != nil {
		t.Fatalf("Failed sending request to %s: %v", ksvc.Status.URL.String(), err)
	}
}
