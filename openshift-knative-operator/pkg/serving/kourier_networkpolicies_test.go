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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	k8stesting "k8s.io/client-go/testing"
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
	manifests, err := generateKourierNetworkPolicies(kourierServing(nil), nil)
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
		"allow-to-tracing",
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

	manifests, err := generateKourierNetworkPolicies(ks, nil)
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

	manifests, err := generateKourierNetworkPolicies(ks, nil)
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

	manifests, err := generateKourierNetworkPolicies(ks, nil)
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

// TestGatewayIngressSources pins the gateway ingress: all sources (the
// OpenShift router and cluster-local traffic from user namespaces) on the
// gateway service targetPorts 8080, 8081, 8443 (https-external) and 8444
// (cluster-local TLS).
func TestGatewayIngressSources(t *testing.T) {
	p := findPolicy(t, kourierNetworkPolicies(kourierNamespace(servingNamespace.Name), nil), "allow-to-kourier-gateway")

	if diff := cmp.Diff(map[string]string{"app": "3scale-kourier-gateway"}, p.Spec.PodSelector.MatchLabels); diff != "" {
		t.Errorf("podSelector diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.Ingress) != 1 {
		t.Fatalf("expected a single ingress rule, got: %+v", p.Spec.Ingress)
	}
	rule := p.Spec.Ingress[0]
	// No From peers at all: an empty NetworkPolicyPeer ({}) is rejected by
	// the API server ("must specify a peer"), omitted From allows all
	// sources, which covers both the OpenShift router and cluster-local
	// traffic from user namespaces.
	if len(rule.From) != 0 {
		t.Errorf("expected no From peers (all sources), got: %+v", rule.From)
	}
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8080))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8081))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8443))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8444))},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

// TestGatewayEgress pins the gateway egress to activator and queue-proxies
// in arbitrary user namespaces: 0.0.0.0/0 on the HTTP (8012), h2c (8013)
// and HTTPS (8112) backend ports.
func TestGatewayEgress(t *testing.T) {
	p := findPolicy(t, kourierNetworkPolicies(kourierNamespace(servingNamespace.Name), nil), "allow-gateway-to-revisions")

	if diff := cmp.Diff(map[string]string{"app": "3scale-kourier-gateway"}, p.Spec.PodSelector.MatchLabels); diff != "" {
		t.Errorf("podSelector diff (-want +got):\n%s", diff)
	}
	if len(p.Spec.Egress) != 1 {
		t.Fatalf("expected a single egress rule, got: %+v", p.Spec.Egress)
	}
	rule := p.Spec.Egress[0]
	assertAnyAddressPeers(t, rule.To)
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8012))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8013))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8112))},
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

// TestKourierMonitoringPorts pins the monitoring ingress in the Kourier
// namespace: openshift-monitoring on 8444 (kube-rbac-proxy, matching the
// other namespaces) and 9090 (the Kourier pods have no kube-rbac-proxy and
// expose their metrics directly).
func TestKourierMonitoringPorts(t *testing.T) {
	p := findPolicy(t, kourierNetworkPolicies(kourierNamespace(servingNamespace.Name), nil), "allow-from-openshift-monitoring")

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
	want := []networkingv1.NetworkPolicyPort{
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8444))},
		{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(9090))},
	}
	if diff := cmp.Diff(want, rule.Ports); diff != "" {
		t.Errorf("ports diff (-want +got):\n%s", diff)
	}
}

func extAuthzServing(extAuthzHost string) *operatorv1beta1.KnativeServing {
	return extAuthzServingWithKey("config-kourier", extAuthzHost)
}

func extAuthzServingWithKey(configKey, extAuthzHost string) *operatorv1beta1.KnativeServing {
	ks := kourierServing(nil)
	if extAuthzHost != "" {
		ks.Spec.CommonSpec.Config = base.ConfigMapData{
			configKey: map[string]string{
				"extauthz-host": extAuthzHost,
			},
		}
	}
	return ks
}

// TestGatewayExtAuthzEgress pins the gateway egress to a configured
// external authorization service (config-kourier extauthz-host): the rule
// is scoped to the authorization service's namespace and port.
func TestGatewayExtAuthzEgress(t *testing.T) {
	ks := extAuthzServing("auth.user-namespace.svc:2222")

	manifests, err := generateKourierNetworkPolicies(ks, nil)
	if err != nil {
		t.Fatalf("generateKourierNetworkPolicies() returned error: %v", err)
	}

	found := false
	for _, m := range manifests {
		for _, r := range m.Resources() {
			if r.GetName() != "allow-gateway-to-extauthz" {
				continue
			}
			found = true
			p := &networkingv1.NetworkPolicy{}
			if err := k8sscheme.Scheme.Convert(&r, p, nil); err != nil {
				t.Fatalf("failed to convert policy: %v", err)
			}
			if diff := cmp.Diff(map[string]string{"app": kourierGatewayPodLabel}, p.Spec.PodSelector.MatchLabels); diff != "" {
				t.Errorf("podSelector diff (-want +got):\n%s", diff)
			}
			if len(p.Spec.Egress) != 1 {
				t.Fatalf("expected a single egress rule, got: %+v", p.Spec.Egress)
			}
			rule := p.Spec.Egress[0]
			if len(rule.To) != 1 || rule.To[0].NamespaceSelector == nil {
				t.Fatalf("expected a namespaceSelector peer, got: %+v", rule.To)
			}
			if diff := cmp.Diff(map[string]string{"kubernetes.io/metadata.name": "user-namespace"}, rule.To[0].NamespaceSelector.MatchLabels); diff != "" {
				t.Errorf("namespaceSelector diff (-want +got):\n%s", diff)
			}
			want := []networkingv1.NetworkPolicyPort{
				{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(2222))},
			}
			if diff := cmp.Diff(want, rule.Ports); diff != "" {
				t.Errorf("ports diff (-want +got):\n%s", diff)
			}
		}
	}
	if !found {
		t.Errorf("expected allow-gateway-to-extauthz NetworkPolicy when extauthz-host is configured")
	}
}

// TestGatewayExtAuthzEgressAbsent pins that no extauthz policy is generated
// when no external authorization service is configured, when the host is a
// bare service name (same namespace, covered by allow-same-namespace), or
// when it is an external hostname that cannot be scoped.
func TestGatewayExtAuthzEgressAbsent(t *testing.T) {
	cases := []struct {
		name     string
		extAuthz string
	}{
		{name: "unset", extAuthz: ""},
		{name: "bare service name", extAuthz: "my-auth:2222"},
		{name: "external hostname", extAuthz: "auth.example.com:2222"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ks := extAuthzServing(c.extAuthz)

			manifests, err := generateKourierNetworkPolicies(ks, nil)
			if err != nil {
				t.Fatalf("generateKourierNetworkPolicies() returned error: %v", err)
			}
			for _, m := range manifests {
				for _, r := range m.Resources() {
					if r.GetName() == "allow-gateway-to-extauthz" {
						t.Errorf("did not expect allow-gateway-to-extauthz NetworkPolicy for extauthz-host %q", c.extAuthz)
					}
				}
			}
		})
	}
}

// TestGatewayExtAuthzEgressShortConfigKey pins that the extauthz
// configuration is also picked up from the spec.config.kourier short key,
// which the operator merges into the config-kourier ConfigMap (the
// "config-" prefix is optional, see ConfigMapTransform).
func TestGatewayExtAuthzEgressShortConfigKey(t *testing.T) {
	ks := extAuthzServingWithKey("kourier", "auth.user-namespace.svc:2222")

	manifests, err := generateKourierNetworkPolicies(ks, nil)
	if err != nil {
		t.Fatalf("generateKourierNetworkPolicies() returned error: %v", err)
	}
	for _, m := range manifests {
		for _, r := range m.Resources() {
			if r.GetName() == "allow-gateway-to-extauthz" {
				return
			}
		}
	}
	t.Errorf("expected allow-gateway-to-extauthz NetworkPolicy for the spec.config.kourier short key")
}

// TestGatewayExtAuthzEgressTargetPort pins that the gateway egress covers
// the authorization service's backend port: OVN-Kubernetes applies egress
// policies after service load balancing (post-DNAT), so the policy must
// allow the targetPort, not only the service port.
func TestGatewayExtAuthzEgressTargetPort(t *testing.T) {
	ks := extAuthzServing("auth.user-namespace.svc:80")

	kube := clientgofake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "user-namespace"},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port:       80,
				TargetPort: intstr.FromInt32(2222),
			}},
		},
	})

	manifests, err := generateKourierNetworkPolicies(ks, kube)
	if err != nil {
		t.Fatalf("generateKourierNetworkPolicies() returned error: %v", err)
	}

	for _, m := range manifests {
		for _, r := range m.Resources() {
			if r.GetName() != "allow-gateway-to-extauthz" {
				continue
			}
			p := &networkingv1.NetworkPolicy{}
			if err := k8sscheme.Scheme.Convert(&r, p, nil); err != nil {
				t.Fatalf("failed to convert policy: %v", err)
			}
			want := []networkingv1.NetworkPolicyPort{
				{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(80))},
				{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(2222))},
			}
			if diff := cmp.Diff(want, p.Spec.Egress[0].Ports); diff != "" {
				t.Errorf("ports diff (-want +got):\n%s", diff)
			}
			return
		}
	}
	t.Fatalf("expected allow-gateway-to-extauthz NetworkPolicy")
}

// TestGatewayExtAuthzEgressNamedTargetPort pins that a named targetPort is
// preserved in the policy: NetworkPolicy ports accept named ports, which
// resolve against the backend pods, so a Service port 80 with
// targetPort: grpc yields the service port plus the named port.
func TestGatewayExtAuthzEgressNamedTargetPort(t *testing.T) {
	ks := extAuthzServing("auth.user-namespace.svc:80")

	kube := clientgofake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "user-namespace"},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port:       80,
				TargetPort: intstr.FromString("grpc"),
			}},
		},
	})

	manifests, err := generateKourierNetworkPolicies(ks, kube)
	if err != nil {
		t.Fatalf("generateKourierNetworkPolicies() returned error: %v", err)
	}

	for _, m := range manifests {
		for _, r := range m.Resources() {
			if r.GetName() != "allow-gateway-to-extauthz" {
				continue
			}
			p := &networkingv1.NetworkPolicy{}
			if err := k8sscheme.Scheme.Convert(&r, p, nil); err != nil {
				t.Fatalf("failed to convert policy: %v", err)
			}
			want := []networkingv1.NetworkPolicyPort{
				{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(80))},
				{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromString("grpc"))},
			}
			if diff := cmp.Diff(want, p.Spec.Egress[0].Ports); diff != "" {
				t.Errorf("ports diff (-want +got):\n%s", diff)
			}
			return
		}
	}
	t.Fatalf("expected allow-gateway-to-extauthz NetworkPolicy")
}

// TestGenerateKourierNetworkPoliciesLookupError pins that transient errors
// from the authorization Service lookup fail the generation, so the
// reconcile retries instead of silently keeping an incomplete policy. A
// missing Service is not an error: the rule falls back to the service port
// and the Service watch re-enqueues the KnativeServing once it appears.
func TestGenerateKourierNetworkPoliciesLookupError(t *testing.T) {
	ks := extAuthzServing("auth.user-namespace.svc:2222")

	kube := clientgofake.NewSimpleClientset()
	kube.PrependReactor("get", "services", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("boom")
	})

	if _, err := generateKourierNetworkPolicies(ks, kube); err == nil {
		t.Errorf("expected generateKourierNetworkPolicies() to return an error on a transient Service lookup failure")
	}
}

// TestKnativeServingsForExtAuthzService pins the mapping used by the Service
// watch: a Service event enqueues exactly the KnativeServings whose
// configured external authorization service it is.
func TestKnativeServingsForExtAuthzService(t *testing.T) {
	withExtAuthz := extAuthzServing("auth.user-namespace.svc:2222")
	withoutExtAuthz := kourierServing(nil)
	otherExtAuthz := extAuthzServing("other.other-namespace.svc:2222")

	got := knativeServingsForExtAuthzService(
		[]*operatorv1beta1.KnativeServing{withExtAuthz, withoutExtAuthz, otherExtAuthz},
		"user-namespace", "auth")
	want := []types.NamespacedName{{Namespace: withExtAuthz.Namespace, Name: withExtAuthz.Name}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("enqueue keys diff (-want +got):\n%s", diff)
	}
}

// TestReconcileDeletesStaleExtAuthzPolicy pins the transition when the
// external authorization service is removed from the configuration: the
// previously applied allow-gateway-to-extauthz policy is deleted (extension
// manifests are not pruned automatically), while an unmanaged policy of the
// same name survives.
func TestReconcileDeletesStaleExtAuthzPolicy(t *testing.T) {
	ks := kourierServing(nil) // no extauthz configured

	ctx, _ := routefake.With(context.Background())
	ctx, _ = configfake.With(ctx, defaultIngress)
	ctx, kube := kubefake.With(ctx, &servingNamespace)
	ext := newFakeExtension(ctx, t)

	kourierNS := kourierNamespace(servingNamespace.Name)
	managed := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-gateway-to-extauthz",
			Namespace: kourierNS,
			Labels:    map[string]string{networkpolicy.ManagedByLabel: "true"},
		},
	}
	// Only one policy per name can exist: seed the managed one and assert
	// deletion first, then an unmanaged one and assert survival.
	if _, err := kube.NetworkingV1().NetworkPolicies(kourierNS).Create(context.Background(), &managed, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed NetworkPolicy: %v", err)
	}

	if err := ext.Reconcile(ctx, ks); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	if _, err := kube.NetworkingV1().NetworkPolicies(kourierNS).Get(context.Background(), "allow-gateway-to-extauthz", metav1.GetOptions{}); err == nil {
		t.Errorf("expected managed 'allow-gateway-to-extauthz' NetworkPolicy to be deleted when extauthz is not configured")
	}

	// An unmanaged policy of the same name must survive the pruning.
	unmanaged := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-gateway-to-extauthz", Namespace: kourierNS},
	}
	if _, err := kube.NetworkingV1().NetworkPolicies(kourierNS).Create(context.Background(), &unmanaged, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed unmanaged NetworkPolicy: %v", err)
	}
	if err := ext.Reconcile(ctx, ks); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}
	if _, err := kube.NetworkingV1().NetworkPolicies(kourierNS).Get(context.Background(), "allow-gateway-to-extauthz", metav1.GetOptions{}); err != nil {
		t.Errorf("expected unmanaged 'allow-gateway-to-extauthz' NetworkPolicy to survive, got: %v", err)
	}
}
