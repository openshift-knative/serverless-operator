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

package networkpolicy

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// TestDenyAll pins the default-deny baseline: both policy types, no rules,
// empty pod selector and the managed-by label used for opt-out deletion.
func TestDenyAll(t *testing.T) {
	p := DenyAll("deny-by-default", "knative-serving")

	if p.Name != "deny-by-default" || p.Namespace != "knative-serving" {
		t.Errorf("unexpected name/namespace: %s/%s", p.Namespace, p.Name)
	}
	if diff := cmp.Diff([]networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}, p.Spec.PolicyTypes); diff != "" {
		t.Errorf("PolicyTypes diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.Ingress) != 0 || len(p.Spec.Egress) != 0 {
		t.Errorf("deny-all must have no rules, got ingress=%v egress=%v", p.Spec.Ingress, p.Spec.Egress)
	}
	if len(p.Spec.PodSelector.MatchLabels) != 0 && len(p.Spec.PodSelector.MatchExpressions) != 0 {
		t.Errorf("deny-all must select all pods, got: %+v", p.Spec.PodSelector)
	}
	if p.Labels[ManagedByLabel] != "true" {
		t.Errorf("expected %s=true label, got: %v", ManagedByLabel, p.Labels)
	}
}

// TestAllowSameNamespace pins same-namespace ingress and egress.
func TestAllowSameNamespace(t *testing.T) {
	p := AllowSameNamespace("allow-same-namespace", "knative-serving")

	wantTypes := []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}
	if diff := cmp.Diff(wantTypes, p.Spec.PolicyTypes); diff != "" {
		t.Errorf("PolicyTypes diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.Ingress) != 1 || len(p.Spec.Ingress[0].From) != 1 || p.Spec.Ingress[0].From[0].PodSelector == nil {
		t.Errorf("expected single ingress rule with same-namespace pod selector, got: %+v", p.Spec.Ingress)
	}
	if len(p.Spec.Egress) != 1 || len(p.Spec.Egress[0].To) != 1 || p.Spec.Egress[0].To[0].PodSelector == nil {
		t.Errorf("expected single egress rule with same-namespace pod selector, got: %+v", p.Spec.Egress)
	}
	if p.Labels[ManagedByLabel] != "true" {
		t.Errorf("expected %s=true label, got: %v", ManagedByLabel, p.Labels)
	}
}

// TestAllowToAPIServer pins the API server egress: 0.0.0.0/0, TCP 443 and 6443
// (6443 is the external API endpoint on HyperShift, 443 the in-cluster service).
func TestAllowToAPIServer(t *testing.T) {
	p := AllowToAPIServer("allow-to-api-server", "knative-serving")

	if len(p.Spec.Egress) != 1 {
		t.Fatalf("expected a single egress rule, got: %+v", p.Spec.Egress)
	}
	rule := p.Spec.Egress[0]
	if len(rule.To) != 1 || rule.To[0].IPBlock == nil || rule.To[0].IPBlock.CIDR != "0.0.0.0/0" {
		t.Errorf("expected ipBlock 0.0.0.0/0, got: %+v", rule.To)
	}
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: portPtr(443)},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: portPtr(6443)},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
	if p.Labels[ManagedByLabel] != "true" {
		t.Errorf("expected %s=true label, got: %v", ManagedByLabel, p.Labels)
	}
}

// TestAllowToDNS pins the DNS egress: openshift-dns namespace on UDP+TCP 53
// and 5353 — all four combinations are required.
func TestAllowToDNS(t *testing.T) {
	p := AllowToDNS("allow-to-dns", "knative-serving")

	if len(p.Spec.Egress) != 1 {
		t.Fatalf("expected a single egress rule, got: %+v", p.Spec.Egress)
	}
	rule := p.Spec.Egress[0]
	if len(rule.To) != 1 || rule.To[0].NamespaceSelector == nil {
		t.Fatalf("expected a namespaceSelector peer, got: %+v", rule.To)
	}
	if diff := cmp.Diff(map[string]string{"kubernetes.io/metadata.name": "openshift-dns"}, rule.To[0].NamespaceSelector.MatchLabels); diff != "" {
		t.Errorf("namespaceSelector diff (-want +got):\n%s", diff)
	}
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolUDP), Port: portPtr(53)},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: portPtr(53)},
		{Protocol: ptr.To(corev1.ProtocolUDP), Port: portPtr(5353)},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: portPtr(5353)},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
	if p.Labels[ManagedByLabel] != "true" {
		t.Errorf("expected %s=true label, got: %v", ManagedByLabel, p.Labels)
	}
}

// TestAllowFromMonitoring pins the prometheus ingress: openshift-monitoring
// namespace, TCP 8444 (kube-rbac-proxy secure metrics port), all pods.
func TestAllowFromMonitoring(t *testing.T) {
	p := AllowFromMonitoring("allow-from-openshift-monitoring", "knative-serving")

	if len(p.Spec.Ingress) != 1 {
		t.Fatalf("expected a single ingress rule, got: %+v", p.Spec.Ingress)
	}
	rule := p.Spec.Ingress[0]
	if len(rule.From) != 1 || rule.From[0].NamespaceSelector == nil {
		t.Fatalf("expected a namespaceSelector peer, got: %+v", rule.From)
	}
	if diff := cmp.Diff(map[string]string{"kubernetes.io/metadata.name": "openshift-monitoring"}, rule.From[0].NamespaceSelector.MatchLabels); diff != "" {
		t.Errorf("namespaceSelector diff (-want +got):\n%s", diff)
	}
	want := []networkingv1.NetworkPolicyPort{{Protocol: ptr.To(corev1.ProtocolTCP), Port: portPtr(8444)}}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.PodSelector.MatchLabels) != 0 && len(p.Spec.PodSelector.MatchExpressions) != 0 {
		t.Errorf("monitoring policy must select all pods, got: %+v", p.Spec.PodSelector)
	}
	if p.Labels[ManagedByLabel] != "true" {
		t.Errorf("expected %s=true label, got: %v", ManagedByLabel, p.Labels)
	}
}

// TestToManifests verifies the conversion to manifestival manifests with the
// NetworkPolicy GVK intact.
func TestToManifests(t *testing.T) {
	policies := []networkingv1.NetworkPolicy{
		DenyAll("deny-by-default", "knative-serving"),
		AllowToDNS("allow-to-dns", "knative-serving"),
	}

	manifests, err := ToManifests(policies)
	if err != nil {
		t.Fatalf("ToManifests() returned error: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected a single manifest, got: %d", len(manifests))
	}
	resources := manifests[0].Resources()
	if len(resources) != len(policies) {
		t.Fatalf("expected %d resources, got: %d", len(policies), len(resources))
	}
	for _, r := range resources {
		if r.GetKind() != "NetworkPolicy" || r.GetAPIVersion() != "networking.k8s.io/v1" {
			t.Errorf("unexpected GVK: %s %s", r.GetAPIVersion(), r.GetKind())
		}
		if r.GetLabels()[ManagedByLabel] != "true" {
			t.Errorf("expected %s=true label on %s, got: %v", ManagedByLabel, r.GetName(), r.GetLabels())
		}
	}
}

func portPtr(p int32) *intstr.IntOrString { return ptr.To(intstr.FromInt32(p)) }

// TestAllowToTracing pins the tracing egress: every pod may export spans to
// a tracing collector (OpenTelemetry 4317/4318, Zipkin 9411). The collector
// namespace is user-chosen, so the rule is port-scoped only.
func TestAllowToTracing(t *testing.T) {
	p := AllowToTracing("allow-to-tracing", "knative-serving")

	if len(p.Spec.Egress) != 1 {
		t.Fatalf("expected a single egress rule, got: %+v", p.Spec.Egress)
	}
	rule := p.Spec.Egress[0]
	if len(rule.To) != 1 || rule.To[0].IPBlock == nil || rule.To[0].IPBlock.CIDR != "0.0.0.0/0" {
		t.Errorf("expected ipBlock 0.0.0.0/0, got: %+v", rule.To)
	}
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: portPtr(4317)},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: portPtr(4318)},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: portPtr(9411)},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.PodSelector.MatchLabels) != 0 && len(p.Spec.PodSelector.MatchExpressions) != 0 {
		t.Errorf("tracing policy must select all pods, got: %+v", p.Spec.PodSelector)
	}
	if p.Labels[ManagedByLabel] != "true" {
		t.Errorf("expected %s=true label, got: %v", ManagedByLabel, p.Labels)
	}
}
