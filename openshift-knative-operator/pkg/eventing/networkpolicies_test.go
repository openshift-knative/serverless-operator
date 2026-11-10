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

package eventing

import (
	"context"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	kubefake "knative.dev/pkg/client/injection/kube/client/fake"
	dynamicfake "knative.dev/pkg/injection/clients/dynamicclient/fake"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/networkpolicy"
)

// TestGenerateDefaultNetworkPolicies pins the complete set of policies
// generated for the knative-eventing namespace.
func TestGenerateDefaultNetworkPolicies(t *testing.T) {
	manifests, err := generateDefaultNetworkPolicies(ke())
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
		"allow-from-openshift-monitoring-metrics",
		"allow-to-tracing",
		"allow-to-webhooks",
		"allow-to-data-plane",
		"allow-to-kafka-data-plane",
		"allow-event-delivery",
	}

	got := []string{}
	for _, r := range manifests[0].Resources() {
		if r.GetNamespace() != requiredNs {
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

// TestWebhookPolicyAcceptsAllSources pins the webhook ingress: the
// eventing-webhook, imc-controller (serving the inmemorychannel-webhook
// service) and kafka-webhook-eventing pods accept admission calls from all
// sources on TCP 8443.
func TestWebhookPolicyAcceptsAllSources(t *testing.T) {
	p := findEventingPolicy(t, eventingPolicies(), "allow-to-webhooks")

	wantSelector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      "app.kubernetes.io/component",
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"eventing-webhook", "imc-controller", "kafka-webhook-eventing"},
		}},
	}
	if diff := cmp.Diff(wantSelector, p.Spec.PodSelector); diff != "" {
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
	wantPorts := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8443))},
	}
	if diff := cmp.Diff(wantPorts, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

// TestDataPlanePolicyAcceptsAllSources pins the ingress for the eventing
// data-plane receivers (broker-ingress, imc-dispatcher, job-sink,
// request-reply, eventmesh-backend): event producers in arbitrary user
// namespaces POST CloudEvents to them on the HTTP (8080) and HTTPS (8443)
// ports; the Backstage eventmesh-backend serves Developer Hub requests on
// 8080.
func TestDataPlanePolicyAcceptsAllSources(t *testing.T) {
	p := findEventingPolicy(t, eventingPolicies(), "allow-to-data-plane")

	wantSelector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      "app.kubernetes.io/component",
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"broker-ingress", "imc-dispatcher", "job-sink", "request-reply", "eventmesh-backend"},
		}},
	}
	if diff := cmp.Diff(wantSelector, p.Spec.PodSelector); diff != "" {
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
	wantPorts := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8080))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8443))},
	}
	if diff := cmp.Diff(wantPorts, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

// TestEventDeliveryEgressUnrestricted pins the delivery egress: the
// data-plane roles that dial subscriber, sink and target URIs (arbitrary user
// namespaces, arbitrary ports) get unrestricted egress. The Kafka
// dispatchers and controller are included: they consume from and produce to
// Kafka bootstrap servers on arbitrary hosts and ports (they run in
// knative-eventing, see eventing-kafka-broker.yaml).
func TestEventDeliveryEgressUnrestricted(t *testing.T) {
	p := findEventingPolicy(t, eventingPolicies(), "allow-event-delivery")

	wantSelector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      "app.kubernetes.io/component",
			Operator: metav1.LabelSelectorOpIn,
			Values: []string{"broker-filter", "imc-dispatcher", "pingsource-mt-adapter", "request-reply",
				"kafka-broker-dispatcher", "kafka-channel-dispatcher", "kafka-source-dispatcher", "kafka-controller"},
		}},
	}
	if diff := cmp.Diff(wantSelector, p.Spec.PodSelector); diff != "" {
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
		t.Errorf("expected no port scoping (subscriber URIs are arbitrary), got: %+v", rule.Ports)
	}
}

// TestGenerateDefaultNetworkPoliciesDisabled verifies the opt-out annotation
// stops eventing policy generation.
func TestGenerateDefaultNetworkPoliciesDisabled(t *testing.T) {
	disabled := ke(func(k *operatorv1beta1.KnativeEventing) {
		k.Annotations = map[string]string{networkpolicy.DisableGenerationAnnotation: "true"}
	})

	manifests, err := generateDefaultNetworkPolicies(disabled)
	if err != nil {
		t.Fatalf("generateDefaultNetworkPolicies() returned error: %v", err)
	}
	if manifests != nil {
		t.Errorf("expected no manifests when generation is disabled, got: %v", manifests)
	}
}

// TestGenerateDefaultNetworkPoliciesSkippedWithMesh verifies that no policies
// are generated when the Service Mesh (istio) integration is enabled: the
// mesh variant with its istiod egress rule is a follow-up (SRVCOM-3716).
func TestGenerateDefaultNetworkPoliciesSkippedWithMesh(t *testing.T) {
	ke := ke(func(k *operatorv1beta1.KnativeEventing) {
		k.Spec.CommonSpec.Config = base.ConfigMapData{
			"features": map[string]string{"istio": "enabled"},
		}
	})

	manifests, err := generateDefaultNetworkPolicies(ke)
	if err != nil {
		t.Fatalf("generateDefaultNetworkPolicies() returned error: %v", err)
	}
	if manifests != nil {
		t.Errorf("expected no manifests when the mesh integration is enabled, got: %v", manifests)
	}
}

func findEventingPolicy(t *testing.T, policies []networkingv1.NetworkPolicy, name string) networkingv1.NetworkPolicy {
	t.Helper()
	for _, p := range policies {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("policy %q not found", name)
	return networkingv1.NetworkPolicy{}
}

// TestManifestsDeleteNetworkPoliciesWhenDisabled verifies the disable
// transition: with the opt-out annotation, Manifests() neither returns the
// policies nor leaves previously generated ones behind.
func TestManifestsDeleteNetworkPoliciesWhenDisabled(t *testing.T) {
	if os.Getenv("KO_DATA_PATH") == "" {
		t.Setenv("KO_DATA_PATH", "../../cmd/openshift-knative-operator/kodata")
	}
	_, filename, _, _ := goruntime.Caller(0)
	t.Setenv("SERVICE_MONITOR_RBAC_MANIFEST_PATH", filepath.Join(filepath.Dir(filename), "..", "monitoring", "testdata", "rbac-proxy.yaml"))

	disabled := ke(func(k *operatorv1beta1.KnativeEventing) {
		k.Annotations = map[string]string{networkpolicy.DisableGenerationAnnotation: "true"}
	})

	ctx, _ := kubefake.With(context.Background(), &eventingNamespace)
	ctx, dynamicClient := dynamicfake.With(ctx, k8sscheme.Scheme)
	ext := NewExtension(ctx, nil).(*extension)

	// Seed a previously generated policy as if generation was enabled before.
	seed := networkpolicy.DenyAll("deny-by-default", requiredNs)
	seedGVR := schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}
	seedU, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&seed)
	if err != nil {
		t.Fatalf("failed to convert seed policy: %v", err)
	}
	if _, err := dynamicClient.Resource(seedGVR).Namespace(requiredNs).Create(context.Background(), &unstructured.Unstructured{Object: seedU}, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed NetworkPolicy: %v", err)
	}

	manifests, err := ext.Manifests(disabled)
	if err != nil {
		t.Fatalf("Manifests() returned error: %v", err)
	}
	for _, m := range manifests {
		for _, r := range m.Resources() {
			if r.GetKind() == "NetworkPolicy" && r.GetLabels()[networkpolicy.ManagedByLabel] == "true" {
				t.Errorf("expected no operand NetworkPolicies in manifests when disabled, got: %s", r.GetName())
			}
		}
	}

	if _, err := dynamicClient.Resource(seedGVR).Namespace(requiredNs).Get(context.Background(), "deny-by-default", metav1.GetOptions{}); err == nil {
		t.Errorf("expected managed 'deny-by-default' NetworkPolicy to be deleted after disabling generation")
	}
}

// eventingPolicies returns the typed eventing policies for the default
// knative-eventing namespace.
func eventingPolicies() []networkingv1.NetworkPolicy {
	return eventingNetworkPolicies(requiredNs)
}

// TestKafkaDataPlanePolicyAcceptsAllSources pins the ingress for the Knative
// Kafka receivers (broker, channel, sink), which run in knative-eventing and
// receive events from arbitrary user namespaces on 8080/8443.
func TestKafkaDataPlanePolicyAcceptsAllSources(t *testing.T) {
	p := findEventingPolicy(t, eventingPolicies(), "allow-to-kafka-data-plane")

	wantSelector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      "app.kubernetes.io/component",
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"kafka-broker-receiver", "kafka-channel-receiver", "kafka-sink-receiver"},
		}},
	}
	if diff := cmp.Diff(wantSelector, p.Spec.PodSelector); diff != "" {
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
	wantPorts := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8080))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8443))},
	}
	if diff := cmp.Diff(wantPorts, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}
