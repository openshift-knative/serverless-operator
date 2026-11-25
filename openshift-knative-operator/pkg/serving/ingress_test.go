package serving

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"

	configfake "github.com/openshift-knative/serverless-operator/pkg/client/config/injection/client/fake"
	routefake "github.com/openshift-knative/serverless-operator/pkg/client/route/injection/client/fake"
	kubefake "knative.dev/pkg/client/injection/kube/client/fake"
)

func TestDefaultIngressClass(t *testing.T) {
	cases := []struct {
		name     string
		in       *operatorv1beta1.KnativeServing
		expected string
	}{{
		name:     "unset",
		in:       &operatorv1beta1.KnativeServing{},
		expected: kourierIngressClassName,
	}, {
		name: "all disabled",
		in: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				Ingress: &operatorv1beta1.IngressConfigs{},
			},
		},
		expected: kourierIngressClassName,
	}, {
		name: "istio enabled",
		in: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				Ingress: &operatorv1beta1.IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: istioIngressClassName,
	}, {
		name: "kourier and istio enabled",
		in: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				Ingress: &operatorv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: kourierIngressClassName,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := defaultIngressClass(c.in)
			if !cmp.Equal(got, c.expected) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", got, c.expected, cmp.Diff(got, c.expected))
			}
		})
	}
}

// TestReconcileEnsuresWebhookNetworkPolicy verifies that, with Istio ingress
// enabled, Reconcile eagerly creates the "webhook" NetworkPolicy so the apiserver
// can reach the validation webhook before the install stage triggers it -
// without it, the Service Mesh deny-all policy deadlocks the install.
func TestReconcileEnsuresWebhookNetworkPolicy(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Namespace: servingNamespace.Name},
		Spec: operatorv1beta1.KnativeServingSpec{
			Ingress: &operatorv1beta1.IngressConfigs{
				Istio: base.IstioIngressConfiguration{Enabled: true},
			},
		},
	}

	ctx, _ := routefake.With(context.Background())
	ctx, _ = configfake.With(ctx, defaultIngress)
	ctx, kube := kubefake.With(ctx, &servingNamespace)
	ext := newFakeExtension(ctx, t)

	if err := ext.Reconcile(ctx, ks); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	np, err := kube.NetworkingV1().NetworkPolicies(servingNamespace.Name).Get(context.Background(), "webhook", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected 'webhook' NetworkPolicy to exist after Reconcile: %v", err)
	}
	if np.Spec.PodSelector.MatchLabels["app"] != "webhook" {
		t.Errorf("expected podSelector app=webhook, got: %+v", np.Spec.PodSelector)
	}
	// Allow-all ingress is a single rule with no "from" peers.
	if len(np.Spec.Ingress) != 1 || len(np.Spec.Ingress[0].From) != 0 {
		t.Errorf("expected a single allow-all ingress rule, got: %+v", np.Spec.Ingress)
	}
}

// TestReconcileSkipsWebhookNetworkPolicyWithoutIstio verifies that with Istio
// ingress disabled (e.g. Kourier), Reconcile creates no istio NetworkPolicies.
func TestReconcileSkipsWebhookNetworkPolicyWithoutIstio(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Namespace: servingNamespace.Name},
		Spec: operatorv1beta1.KnativeServingSpec{
			Ingress: &operatorv1beta1.IngressConfigs{
				Kourier: base.KourierIngressConfiguration{Enabled: true},
			},
		},
	}

	ctx, _ := routefake.With(context.Background())
	ctx, _ = configfake.With(ctx, defaultIngress)
	ctx, kube := kubefake.With(ctx, &servingNamespace)
	ext := newFakeExtension(ctx, t)

	if err := ext.Reconcile(ctx, ks); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	if _, err := kube.NetworkingV1().NetworkPolicies(servingNamespace.Name).Get(context.Background(), "webhook", metav1.GetOptions{}); err == nil {
		t.Errorf("did not expect 'webhook' NetworkPolicy to be created when Istio ingress is disabled")
	}
}
