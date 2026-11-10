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

// TestGenerateDefaultNetworkPolicies pins the complete set of policies
// generated for the knative-serving namespace.
func TestGenerateDefaultNetworkPolicies(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Namespace: servingNamespace.Name},
	}

	manifests, err := generateDefaultNetworkPolicies(ks)
	if err != nil {
		t.Fatalf("generateDefaultNetworkPolicies() returned error: %v", err)
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
		"allow-to-tracing",
		"allow-to-webhook",
		"allow-from-kourier-gateway-to-activator",
		"allow-activator-to-revisions",
		"allow-autoscaler-to-revision-metrics",
		"allow-controller-to-registries",
	}

	got := []string{}
	for _, r := range manifests[0].Resources() {
		if r.GetNamespace() != servingNamespace.Name {
			t.Errorf("policy %s in wrong namespace: %s", r.GetName(), r.GetNamespace())
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

// TestGenerateDefaultNetworkPoliciesDisabled verifies the opt-out annotation
// stops policy generation.
func TestGenerateDefaultNetworkPoliciesDisabled(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: servingNamespace.Name,
			Annotations: map[string]string{
				networkpolicy.DisableGenerationAnnotation: "true",
			},
		},
	}

	manifests, err := generateDefaultNetworkPolicies(ks)
	if err != nil {
		t.Fatalf("generateDefaultNetworkPolicies() returned error: %v", err)
	}
	if manifests != nil {
		t.Errorf("expected no manifests when generation is disabled, got: %v", manifests)
	}
}

func findPolicy(t *testing.T, policies []networkingv1.NetworkPolicy, name string) networkingv1.NetworkPolicy {
	t.Helper()
	for _, p := range policies {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("policy %q not found", name)
	return networkingv1.NetworkPolicy{}
}

// TestWebhookPolicyAcceptsAllSources pins the webhook ingress: all sources
// (admission calls can originate off-cluster, e.g. on HyperShift), TCP 8443.
func TestWebhookPolicyAcceptsAllSources(t *testing.T) {
	p := findPolicy(t, servingNetworkPolicies(servingNamespace.Name), "allow-to-webhook")

	if diff := cmp.Diff(map[string]string{"app": "webhook"}, p.Spec.PodSelector.MatchLabels); diff != "" {
		t.Errorf("podSelector diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.Ingress) != 1 {
		t.Fatalf("expected a single ingress rule, got: %+v", p.Spec.Ingress)
	}
	rule := p.Spec.Ingress[0]
	// No From peers at all: an empty NetworkPolicyPeer ({}) is rejected by
	// the API server ("must specify a peer"), omitted From allows all sources.
	if len(rule.From) != 0 {
		t.Errorf("expected no From peers (all sources), got: %+v", rule.From)
	}
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8443))},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

// TestActivatorEgressUnrestricted pins the activator egress to revision pods
// in arbitrary user namespaces: 0.0.0.0/0 on the HTTP (8012), h2c (8013) and
// HTTPS (8112) backend ports.
func TestActivatorEgressUnrestricted(t *testing.T) {
	p := findPolicy(t, servingNetworkPolicies(servingNamespace.Name), "allow-activator-to-revisions")

	if diff := cmp.Diff(map[string]string{"app": "activator"}, p.Spec.PodSelector.MatchLabels); diff != "" {
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
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8013))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8112))},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

// TestAutoscalerEgressRevisionMetrics pins the autoscaler scrape egress to
// queue-proxies in arbitrary user namespaces: 0.0.0.0/0 on TCP 9090 only.
func TestAutoscalerEgressRevisionMetrics(t *testing.T) {
	p := findPolicy(t, servingNetworkPolicies(servingNamespace.Name), "allow-autoscaler-to-revision-metrics")

	if diff := cmp.Diff(map[string]string{"app": "autoscaler"}, p.Spec.PodSelector.MatchLabels); diff != "" {
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
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(9090))},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

// TestGatewayToActivatorIngress pins the Kourier gateway -> activator cold
// start path: ingress from the kourier (-ingress) namespace on the HTTP
// (8012), h2c (8013) and HTTPS (8112) backend ports.
func TestGatewayToActivatorIngress(t *testing.T) {
	p := findPolicy(t, servingNetworkPolicies(servingNamespace.Name), "allow-from-kourier-gateway-to-activator")

	if diff := cmp.Diff(map[string]string{"app": "activator"}, p.Spec.PodSelector.MatchLabels); diff != "" {
		t.Errorf("podSelector diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.Ingress) != 1 {
		t.Fatalf("expected a single ingress rule, got: %+v", p.Spec.Ingress)
	}
	rule := p.Spec.Ingress[0]
	if len(rule.From) != 1 || rule.From[0].NamespaceSelector == nil {
		t.Fatalf("expected a namespaceSelector peer, got: %+v", rule.From)
	}
	wantNS := kourierNamespace(servingNamespace.Name)
	if diff := cmp.Diff(map[string]string{"kubernetes.io/metadata.name": wantNS}, rule.From[0].NamespaceSelector.MatchLabels); diff != "" {
		t.Errorf("namespaceSelector diff (-want +got):\n%s", diff)
	}
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8012))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8013))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8112))},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

// TestControllerEgressToRegistries pins the controller egress for tag to
// digest resolution against container registries, which use arbitrary hosts
// and ports (e.g. the internal image registry on 5000): unrestricted.
func TestControllerEgressToRegistries(t *testing.T) {
	p := findPolicy(t, servingNetworkPolicies(servingNamespace.Name), "allow-controller-to-registries")

	if diff := cmp.Diff(map[string]string{"app": "controller"}, p.Spec.PodSelector.MatchLabels); diff != "" {
		t.Errorf("podSelector diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.Egress) != 1 {
		t.Fatalf("expected a single egress rule, got: %+v", p.Spec.Egress)
	}
	rule := p.Spec.Egress[0]
	if len(rule.To) != 1 || rule.To[0].IPBlock == nil || rule.To[0].IPBlock.CIDR != "0.0.0.0/0" {
		t.Errorf("expected ipBlock 0.0.0.0/0, got: %+v", rule.To)
	}
	if len(rule.Ports) != 0 {
		t.Errorf("expected no port scoping (registry endpoints are arbitrary), got: %+v", rule.Ports)
	}
}

// TestGenerateDefaultNetworkPoliciesSkippedWithIstio verifies that no
// policies are generated for the knative-serving namespace when the Istio
// ingress (Service Mesh) integration is enabled: the mesh variant with its
// istiod egress rule is a follow-up of SRVCOM-3716.
func TestGenerateDefaultNetworkPoliciesSkippedWithIstio(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Namespace: servingNamespace.Name},
		Spec: operatorv1beta1.KnativeServingSpec{
			Ingress: &operatorv1beta1.IngressConfigs{
				Istio: base.IstioIngressConfiguration{Enabled: true},
			},
		},
	}

	manifests, err := generateDefaultNetworkPolicies(ks)
	if err != nil {
		t.Fatalf("generateDefaultNetworkPolicies() returned error: %v", err)
	}
	if manifests != nil {
		t.Errorf("expected no manifests when the Istio ingress is enabled, got: %v", manifests)
	}
}

// TestReconcileDeletesNetworkPoliciesWhenDisabled verifies the disable
// transition: setting the opt-out annotation on an existing install deletes
// the previously generated policies rather than freezing them.
func TestReconcileDeletesNetworkPoliciesWhenDisabled(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: servingNamespace.Name,
			Annotations: map[string]string{
				networkpolicy.DisableGenerationAnnotation: "true",
			},
		},
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

	// Seed a previously generated policy as if generation was enabled before.
	seed := networkpolicy.DenyAll("deny-by-default", servingNamespace.Name)
	if _, err := kube.NetworkingV1().NetworkPolicies(servingNamespace.Name).Create(context.Background(), &seed, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed NetworkPolicy: %v", err)
	}
	unmanaged := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "user-managed", Namespace: servingNamespace.Name},
	}
	if _, err := kube.NetworkingV1().NetworkPolicies(servingNamespace.Name).Create(context.Background(), &unmanaged, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed unmanaged NetworkPolicy: %v", err)
	}

	if err := ext.Reconcile(ctx, ks); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	if _, err := kube.NetworkingV1().NetworkPolicies(servingNamespace.Name).Get(context.Background(), "deny-by-default", metav1.GetOptions{}); err == nil {
		t.Errorf("expected managed 'deny-by-default' NetworkPolicy to be deleted after disabling generation")
	}
	if _, err := kube.NetworkingV1().NetworkPolicies(servingNamespace.Name).Get(context.Background(), "user-managed", metav1.GetOptions{}); err != nil {
		t.Errorf("expected unmanaged 'user-managed' NetworkPolicy to survive, got: %v", err)
	}
}

// TestFinalizeDeletesNetworkPolicies verifies that deleting the
// KnativeServing CR cleans up the operand NetworkPolicies in both managed
// namespaces: the Kourier-namespace policies carry no owner references, so
// neither the garbage collector nor the manifest uninstall removes them.
func TestFinalizeDeletesNetworkPolicies(t *testing.T) {
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

	kourierNS := kourierNamespace(servingNamespace.Name)
	for _, ns := range []string{servingNamespace.Name, kourierNS} {
		seed := networkpolicy.DenyAll("deny-by-default", ns)
		if _, err := kube.NetworkingV1().NetworkPolicies(ns).Create(context.Background(), &seed, metav1.CreateOptions{}); err != nil {
			t.Fatalf("failed to seed NetworkPolicy in %s: %v", ns, err)
		}
	}

	if err := ext.Finalize(ctx, ks); err != nil {
		t.Fatalf("Finalize() returned error: %v", err)
	}

	for _, ns := range []string{servingNamespace.Name, kourierNS} {
		if _, err := kube.NetworkingV1().NetworkPolicies(ns).Get(context.Background(), "deny-by-default", metav1.GetOptions{}); err == nil {
			t.Errorf("expected 'deny-by-default' NetworkPolicy in %s to be deleted on Finalize", ns)
		}
	}
}

// TestManifestsIncludeDefaultNetworkPolicies verifies the policies are part
// of the extension manifests.
func TestManifestsIncludeDefaultNetworkPolicies(t *testing.T) {
	t.Setenv("SERVICE_MONITOR_RBAC_MANIFEST_PATH", "../monitoring/testdata/rbac-proxy.yaml")

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
	ctx, _ = kubefake.With(ctx, &servingNamespace)
	ext := newFakeExtension(ctx, t)

	manifests, err := ext.Manifests(ks)
	if err != nil {
		t.Fatalf("Manifests() returned error: %v", err)
	}

	found := false
	for _, m := range manifests {
		for _, r := range m.Resources() {
			if r.GetName() == "deny-by-default" && r.GetNamespace() == servingNamespace.Name {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected deny-by-default NetworkPolicy in extension manifests")
	}
}
