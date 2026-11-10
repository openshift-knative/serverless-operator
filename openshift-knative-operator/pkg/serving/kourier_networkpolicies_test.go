/*
Copyright 2026 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package serving

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/networkpolicy"
	configfake "github.com/openshift-knative/serverless-operator/pkg/client/config/injection/client/fake"
	routefake "github.com/openshift-knative/serverless-operator/pkg/client/route/injection/client/fake"
	kubefake "knative.dev/pkg/client/injection/kube/client/fake"
)

func kourierServing(annotations map[string]string) *operatorv1beta1.KnativeServing {
	return &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   servingNamespace.Name,
			Name:        "test",
			Annotations: annotations,
		},
		Spec: operatorv1beta1.KnativeServingSpec{
			Ingress: &operatorv1beta1.IngressConfigs{
				Kourier: base.KourierIngressConfiguration{Enabled: true},
			},
		},
	}
}

// TestGenerateKourierNetworkPolicies pins the complete set of policies
// generated for the knative-serving-ingress (Kourier) namespace.
func TestGenerateKourierNetworkPolicies(t *testing.T) {
	manifests, err := generateKourierNetworkPolicies(kourierServing(nil))
	if err != nil {
		t.Fatalf("generateKourierNetworkPolicies() returned error: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected a single manifest, got: %d", len(manifests))
	}

	want := []string{
		"deny-by-default",
		"allow-same-namespace",
		"allow-to-api-server",
		"allow-to-dns",
		"allow-from-openshift-monitoring",
		"allow-to-kourier-gateway",
		"allow-gateway-to-revisions",
	}

	got := []string{}
	for _, r := range manifests[0].Resources() {
		// Pre-transform the policies target the Kourier namespace directly;
		// the provider label lets overrideKourierNamespace move them along
		// with the other Kourier resources.
		if r.GetLabels()[providerLabel] != "kourier" {
			t.Errorf("policy %s missing %s label", r.GetName(), providerLabel)
		}
		if r.GetLabels()[networkpolicy.ManagedByLabel] != "true" {
			t.Errorf("policy %s missing %s label", r.GetName(), networkpolicy.ManagedByLabel)
		}
		got = append(got, r.GetName())
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("policy names diff (-want +got):\n%s", diff)
	}
}

// TestGenerateKourierNetworkPoliciesDisabled verifies the opt-out annotation
// stops Kourier namespace policy generation.
func TestGenerateKourierNetworkPoliciesDisabled(t *testing.T) {
	ks := kourierServing(map[string]string{networkpolicy.DisableGenerationAnnotation: "true"})

	manifests, err := generateKourierNetworkPolicies(ks)
	if err != nil {
		t.Fatalf("generateKourierNetworkPolicies() returned error: %v", err)
	}
	if manifests != nil {
		t.Errorf("expected no manifests when generation is disabled, got: %v", manifests)
	}
}

// TestGenerateKourierNetworkPoliciesSkippedWithoutKourier verifies that no
// Kourier namespace policies are generated when another ingress (e.g.
// Istio) is enabled: the knative-serving-ingress namespace does not exist in
// that setup.
func TestGenerateKourierNetworkPoliciesSkippedWithoutKourier(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Namespace: servingNamespace.Name, Name: "test"},
		Spec: operatorv1beta1.KnativeServingSpec{
			Ingress: &operatorv1beta1.IngressConfigs{
				Istio: base.IstioIngressConfiguration{Enabled: true},
			},
		},
	}

	manifests, err := generateKourierNetworkPolicies(ks)
	if err != nil {
		t.Fatalf("generateKourierNetworkPolicies() returned error: %v", err)
	}
	if manifests != nil {
		t.Errorf("expected no manifests when Kourier is disabled, got: %v", manifests)
	}
}

// TestKourierPoliciesNoOwnerRefs verifies that the Kourier namespace
// policies ride the overrideKourierNamespace transformer: after it runs they
// live in the -ingress namespace without OwnerReferences (a cross-namespace
// owner ref would make the garbage collector delete them).
func TestKourierPoliciesNoOwnerRefs(t *testing.T) {
	ks := kourierServing(nil)

	manifests, err := generateKourierNetworkPolicies(ks)
	if err != nil {
		t.Fatalf("generateKourierNetworkPolicies() returned error: %v", err)
	}

	wantNS := kourierNamespace(servingNamespace.Name)
	for _, m := range manifests {
		// Simulate the owner ref the operator injects into namespaced
		// resources before the extension transformers run.
		for _, r := range m.Resources() {
			r.SetOwnerReferences([]metav1.OwnerReference{{
				APIVersion: "operator.knative.dev/v1beta1",
				Kind:       "KnativeServing",
				Name:       "test",
			}})
		}
		if _, err := m.Transform(overrideKourierNamespace(ks)); err != nil {
			t.Fatalf("Transform() returned error: %v", err)
		}
		for _, r := range m.Resources() {
			if r.GetNamespace() != wantNS {
				t.Errorf("policy %s in wrong namespace after transform: %s, want: %s", r.GetName(), r.GetNamespace(), wantNS)
			}
			if len(r.GetOwnerReferences()) != 0 {
				t.Errorf("policy %s still has owner references after transform: %v", r.GetName(), r.GetOwnerReferences())
			}
		}
	}
}

// TestGatewayIngressSources pins the gateway ingress: the OpenShift router
// (policy-group network label) plus all namespaces for cluster-local
// traffic, on the gateway service targetPorts 8080 and 8081.
func TestGatewayIngressSources(t *testing.T) {
	p := findPolicy(t, kourierNetworkPolicies(kourierNamespace(servingNamespace.Name)), "allow-to-kourier-gateway")

	if diff := cmp.Diff(map[string]string{"app": "3scale-kourier-gateway"}, p.Spec.PodSelector.MatchLabels); diff != "" {
		t.Errorf("podSelector diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.Ingress) != 1 {
		t.Fatalf("expected a single ingress rule, got: %+v", p.Spec.Ingress)
	}
	rule := p.Spec.Ingress[0]
	if len(rule.From) != 2 {
		t.Fatalf("expected two From peers, got: %+v", rule.From)
	}
	hasRouter, hasAll := false, false
	for _, f := range rule.From {
		if f.NamespaceSelector != nil && f.NamespaceSelector.MatchLabels["policy-group.network.openshift.io/ingress"] == "" {
			if _, ok := f.NamespaceSelector.MatchLabels["policy-group.network.openshift.io/ingress"]; ok {
				hasRouter = true
			}
		}
		if f.PodSelector == nil && f.NamespaceSelector == nil && f.IPBlock == nil {
			hasAll = true
		}
	}
	if !hasRouter {
		t.Errorf("expected a router namespaceSelector peer (policy-group.network.openshift.io/ingress), got: %+v", rule.From)
	}
	if !hasAll {
		t.Errorf("expected an empty From peer (all sources, cluster-local traffic), got: %+v", rule.From)
	}
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8080))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8081))},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

// TestGatewayEgress pins the gateway egress to activator and queue-proxies
// in arbitrary user namespaces: 0.0.0.0/0 on TCP 8012 only.
func TestGatewayEgress(t *testing.T) {
	p := findPolicy(t, kourierNetworkPolicies(kourierNamespace(servingNamespace.Name)), "allow-gateway-to-revisions")

	if diff := cmp.Diff(map[string]string{"app": "3scale-kourier-gateway"}, p.Spec.PodSelector.MatchLabels); diff != "" {
		t.Errorf("podSelector diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.Egress) != 1 {
		t.Fatalf("expected a single egress rule, got: %+v", p.Spec.Egress)
	}
	rule := p.Spec.Egress[0]
	if len(rule.To) != 1 || rule.To[0].IPBlock == nil || rule.To[0].IPBlock.CIDR != "0.0.0.0/0" {
		t.Errorf("expected ipBlock 0.0.0.0/0, got: %+v", rule.To)
	}
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8012))},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

// TestManifestsIncludeKourierNetworkPolicies verifies the Kourier policies
// are part of the extension manifests when Kourier is enabled.
func TestManifestsIncludeKourierNetworkPolicies(t *testing.T) {
	t.Setenv("SERVICE_MONITOR_RBAC_MANIFEST_PATH", "../monitoring/testdata/rbac-proxy.yaml")

	ks := kourierServing(nil)

	ctx, _ := routefake.With(context.Background())
	ctx, _ = configfake.With(ctx, defaultIngress)
	ctx, _ = kubefake.With(ctx, &servingNamespace)
	ext := newFakeExtension(ctx, t)

	manifests, err := ext.Manifests(ks)
	if err != nil {
		t.Fatalf("Manifests() returned error: %v", err)
	}

	found := false
	for _, m := range manifests {
		for _, r := range m.Resources() {
			if r.GetName() == "allow-to-kourier-gateway" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected allow-to-kourier-gateway NetworkPolicy in extension manifests")
	}
}
