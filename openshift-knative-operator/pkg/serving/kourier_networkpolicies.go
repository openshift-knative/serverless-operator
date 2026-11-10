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
	mf "github.com/manifestival/manifestival"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/networkpolicy"
)

const (
	// kourierGatewayPodLabel selects the Envoy gateway deployment.
	kourierGatewayPodLabel = "3scale-kourier-gateway"

	// routerNamespaceSelectorLabel is the namespace label OVN-Kubernetes
	// classifies host-network OpenShift router traffic under for NetworkPolicy
	// evaluation.
	routerNamespaceSelectorLabel = "policy-group.network.openshift.io/ingress"

	// kourierGatewayHTTPPort and kourierGatewayInternalPort are the gateway
	// service targetPorts for external and cluster-local traffic.
	kourierGatewayHTTPPort     int32 = 8080
	kourierGatewayInternalPort int32 = 8081
)

// generateKourierNetworkPolicies returns the NetworkPolicies for the Kourier
// namespace (knative-serving-ingress), or nil when generation is disabled
// via the opt-out annotation or when Kourier is not the enabled ingress (the
// namespace only exists with Kourier).
//
// The policies carry the Kourier ingress-provider label, so the
// overrideKourierNamespace transformer moves them into the Kourier namespace
// and strips owner references along with the other Kourier resources.
func generateKourierNetworkPolicies(ks base.KComponent) ([]mf.Manifest, error) {
	if networkPoliciesDisabled(ks) || !kourierIngressEnabled(ks) {
		return nil, nil
	}
	return networkpolicy.ToManifests(kourierNetworkPolicies(kourierNamespace(ks.GetNamespace())))
}

// kourierIngressEnabled reports whether the Kourier ingress is (or defaults
// to being) enabled for the given KnativeServing.
func kourierIngressEnabled(ks base.KComponent) bool {
	kcomponent := ks.(*operatorv1beta1.KnativeServing)
	if kcomponent.Spec.Ingress == nil {
		// Nothing configured defaults to Kourier (see defaultToKourier).
		return true
	}
	return kcomponent.Spec.Ingress.Kourier.Enabled
}

// kourierNetworkPolicies builds the policies isolating the Kourier
// (knative-serving-ingress) namespace: default-deny plus the tailored allows
// for the Envoy gateway data plane (see SRVCOM-3716).
func kourierNetworkPolicies(namespace string) []networkingv1.NetworkPolicy {
	// All policies carry the Kourier provider label so the
	// overrideKourierNamespace transformer moves them into the Kourier
	// namespace and strips owner references (a cross-namespace owner ref
	// would make the garbage collector delete them).
	policies := []networkingv1.NetworkPolicy{
		networkpolicy.DenyAll("deny-by-default", namespace),
		networkpolicy.AllowSameNamespace("allow-same-namespace", namespace),
		networkpolicy.AllowToAPIServer("allow-to-api-server", namespace),
		networkpolicy.AllowToDNS("allow-to-dns", namespace),
		networkpolicy.AllowFromMonitoring("allow-from-openshift-monitoring", namespace),
	}
	for i := range policies {
		policies[i].Labels[providerLabel] = "kourier"
	}

	// The gateway receives external traffic from the OpenShift router and
	// cluster-local traffic from arbitrary user namespaces (ksvc URLs resolve
	// to kourier-internal).
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-to-kourier-gateway",
			Namespace: namespace,
			Labels:    kourierLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": kourierGatewayPodLabel}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								routerNamespaceSelectorLabel: "",
							},
						},
					},
					// All sources: cluster-local traffic from user namespaces.
					{},
				},
				Ports: networkPolicyPorts(
					networkPolicyPortTCP(kourierGatewayHTTPPort),
					networkPolicyPortTCP(kourierGatewayInternalPort),
				),
			}},
		},
	})

	// Gateway -> activator and queue-proxies of revisions in arbitrary user
	// namespaces.
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-gateway-to-revisions",
			Namespace: namespace,
			Labels:    kourierLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": kourierGatewayPodLabel}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: []networkingv1.NetworkPolicyPeer{{
					IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
				}},
				Ports: networkPolicyPorts(networkPolicyPortTCP(activatorPort)),
			}},
		},
	})

	return policies
}

func kourierLabels() map[string]string {
	labels := managedLabels()
	labels[providerLabel] = "kourier"
	return labels
}
