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
	"strconv"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/networkpolicy"
)

const (
	// webhookPort is the targetPort of the serving webhook service.
	webhookPort int32 = 8443
	// activatorPort, activatorHTTP2Port and activatorHTTPSPort are the
	// revision backend ports (knative.dev/networking constants): HTTP/1,
	// h2c and HTTPS respectively, used by the activator, the Kourier
	// gateway and the queue-proxies.
	activatorPort      int32 = 8012
	activatorHTTP2Port int32 = 8013
	activatorHTTPSPort int32 = 8112
	// queueProxyMetricsPort is the queue-proxy metrics port scraped by the
	// autoscaler.
	queueProxyMetricsPort int32 = 9090
)

// generateDefaultNetworkPolicies returns the default-deny NetworkPolicies
// for the knative-serving namespace, or nil when generation is disabled via
// the serverless.openshift.io/disable-network-policies-generation annotation
// or when the Istio ingress (Service Mesh) integration is enabled — the mesh
// variant, which additionally needs an istiod egress rule, is a follow-up of
// SRVCOM-3716.
func generateDefaultNetworkPolicies(ks base.KComponent) ([]mf.Manifest, error) {
	if operandNetworkPoliciesDisabled(ks) {
		return nil, nil
	}
	return networkpolicy.ToManifests(servingNetworkPolicies(ks.GetNamespace()))
}

// operandNetworkPoliciesDisabled reports whether operand NetworkPolicy
// generation must not run for KnativeServing: opted out via the annotation,
// or the Istio ingress (Service Mesh) is enabled.
func operandNetworkPoliciesDisabled(ks base.KComponent) bool {
	return networkPoliciesDisabled(ks) || istioIngressEnabled(ks)
}

// istioIngressEnabled reports whether the Istio ingress is enabled for the
// given KnativeServing, which implies the Service Mesh integration.
func istioIngressEnabled(ks base.KComponent) bool {
	kcomponent, ok := ks.(*operatorv1beta1.KnativeServing)
	if !ok || kcomponent.Spec.Ingress == nil {
		return false
	}
	return kcomponent.Spec.Ingress.Istio.Enabled
}

// networkPoliciesDisabled reports whether the opt-out annotation disables
// operand NetworkPolicy generation.
func networkPoliciesDisabled(ks base.KComponent) bool {
	if v, ok := ks.GetAnnotations()[networkpolicy.DisableGenerationAnnotation]; ok {
		if disable, _ := strconv.ParseBool(v); disable {
			return true
		}
	}
	return false
}

// servingNetworkPolicies builds the policies isolating the knative-serving
// namespace: default-deny plus the tailored allows for the control and data
// plane (see SRVCOM-3716).
func servingNetworkPolicies(namespace string) []networkingv1.NetworkPolicy {
	policies := []networkingv1.NetworkPolicy{
		networkpolicy.DenyAll("deny-by-default", namespace),
		networkpolicy.AllowSameNamespace("allow-same-namespace", namespace),
		networkpolicy.AllowToAPIServer("allow-to-api-server", namespace),
		networkpolicy.AllowToDNS("allow-to-dns", namespace),
		networkpolicy.AllowFromMonitoring("allow-from-openshift-monitoring", namespace),
		networkpolicy.AllowToTracing("allow-to-tracing", namespace),
	}

	// The webhook must be reachable by the apiserver for admission; on
	// HyperShift the apiserver is outside the cluster network, so the rule
	// allows all sources (no From peers).
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-to-webhook",
			Namespace: namespace,
			Labels:    managedLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "webhook"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				Ports: networkPolicyPorts(networkPolicyPortTCP(webhookPort)),
			}},
		},
	})

	// Kourier gateway -> activator: cold start request path.
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-from-kourier-gateway-to-activator",
			Namespace: namespace,
			Labels:    managedLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "activator"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": kourierNamespace(namespace),
						},
					},
				}},
				Ports: networkPolicyPorts(
					networkPolicyPortTCP(activatorPort),
					networkPolicyPortTCP(activatorHTTP2Port),
					networkPolicyPortTCP(activatorHTTPSPort),
				),
			}},
		},
	})

	// Activator -> queue-proxies of revisions in arbitrary user namespaces
	// on the HTTP, h2c and HTTPS backend ports.
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-activator-to-revisions",
			Namespace: namespace,
			Labels:    managedLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "activator"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: networkpolicy.AnyAddressPeers(),
				Ports: networkPolicyPorts(
					networkPolicyPortTCP(activatorPort),
					networkPolicyPortTCP(activatorHTTP2Port),
					networkPolicyPortTCP(activatorHTTPSPort),
				),
			}},
		},
	})

	// Autoscaler -> queue-proxy metrics of revisions in arbitrary user
	// namespaces.
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-autoscaler-to-revision-metrics",
			Namespace: namespace,
			Labels:    managedLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "autoscaler"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To:    networkpolicy.AnyAddressPeers(),
				Ports: networkPolicyPorts(networkPolicyPortTCP(queueProxyMetricsPort)),
			}},
		},
	})

	// Controller -> container registries for tag to digest resolution:
	// registry hosts and ports are arbitrary (e.g. the internal OpenShift
	// image registry on 5000), so the egress is unrestricted.
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-controller-to-registries",
			Namespace: namespace,
			Labels:    managedLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "controller"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: networkpolicy.AnyAddressPeers(),
			}},
		},
	})

	return policies
}

func managedLabels() map[string]string {
	return map[string]string{networkpolicy.ManagedByLabel: "true"}
}

func networkPolicyPortTCP(port int32) networkingv1.NetworkPolicyPort {
	return networkingv1.NetworkPolicyPort{
		Protocol: ptr.To(corev1.ProtocolTCP),
		Port:     ptr.To(intstr.FromInt32(port)),
	}
}

func networkPolicyPorts(ports ...networkingv1.NetworkPolicyPort) []networkingv1.NetworkPolicyPort {
	return ports
}
