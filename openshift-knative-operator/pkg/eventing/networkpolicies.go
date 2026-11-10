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
	"strconv"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"knative.dev/operator/pkg/apis/operator/base"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/networkpolicy"
	"github.com/openshift-knative/serverless-operator/pkg/istio/eventingistio"
)

const (
	// componentLabel is the pod label all eventing deployments carry, with
	// the component name as value (eventing-webhook, broker-filter, ...).
	componentLabel = "app.kubernetes.io/component"

	// webhookPort is the targetPort of the eventing webhook services
	// (eventing-webhook, inmemorychannel-webhook).
	webhookPort int32 = 8443

	// dataPlaneHTTPPort and dataPlaneHTTPSPort are the targetPorts of the
	// eventing data-plane receiver services (broker-ingress, imc-dispatcher,
	// job-sink, request-reply).
	dataPlaneHTTPPort  int32 = 8080
	dataPlaneHTTPSPort int32 = 8443
)

// generateDefaultNetworkPolicies returns the default-deny NetworkPolicies
// for the knative-eventing namespace, or nil when generation is disabled via
// the serverless.openshift.io/disable-network-policies-generation annotation
// or when the Service Mesh (istio) integration is enabled — the mesh variant,
// which additionally needs an istiod egress rule, is a follow-up of
// SRVCOM-3716.
func generateDefaultNetworkPolicies(ke base.KComponent) ([]mf.Manifest, error) {
	if eventingNetworkPoliciesDisabled(ke) {
		return nil, nil
	}
	return networkpolicy.ToManifests(eventingNetworkPolicies(ke.GetNamespace()))
}

// operandNetworkPoliciesForDeletion returns the manifests of the operand
// NetworkPolicies that would have been generated, for deletion when
// generation is disabled (opt-out annotation or mesh enabled). Deleting the
// would-be resources by their deterministic names handles the enable ->
// disable transition, mirroring the Service Mesh policy handling.
func operandNetworkPoliciesForDeletion(ke base.KComponent) []mf.Manifest {
	manifests, err := networkpolicy.ToManifests(eventingNetworkPolicies(ke.GetNamespace()))
	if err != nil {
		return nil
	}
	return manifests
}

// eventingNetworkPoliciesDisabled reports whether operand NetworkPolicy
// generation must not run: opted out via the annotation, or the Service Mesh
// integration is enabled (features.istio=enabled).
func eventingNetworkPoliciesDisabled(ke base.KComponent) bool {
	if v, ok := ke.GetAnnotations()[networkpolicy.DisableGenerationAnnotation]; ok {
		if disable, _ := strconv.ParseBool(v); disable {
			return true
		}
	}
	return eventingistio.IsEnabled(ke.GetSpec().GetConfig())
}

// eventingNetworkPolicies builds the policies isolating the knative-eventing
// namespace: default-deny plus the tailored allows for the control and data
// plane (see SRVCOM-3716).
func eventingNetworkPolicies(namespace string) []networkingv1.NetworkPolicy {
	policies := []networkingv1.NetworkPolicy{
		networkpolicy.DenyAll("deny-by-default", namespace),
		networkpolicy.AllowSameNamespace("allow-same-namespace", namespace),
		networkpolicy.AllowToAPIServer("allow-to-api-server", namespace),
		networkpolicy.AllowToDNS("allow-to-dns", namespace),
		networkpolicy.AllowFromMonitoring("allow-from-openshift-monitoring", namespace),
	}

	// The webhooks must be reachable by the apiserver for admission; on
	// HyperShift the apiserver is outside the cluster network, so the rule
	// allows all sources. The inmemorychannel-webhook service is served by
	// the imc-controller pods.
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-to-webhooks",
			Namespace: namespace,
			Labels:    managedLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: componentSelector("eventing-webhook", "imc-controller"),
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From:  []networkingv1.NetworkPolicyPeer{{}},
				Ports: networkPolicyPorts(networkPolicyPortTCP(webhookPort)),
			}},
		},
	})

	// Data-plane receivers: event producers in arbitrary user namespaces
	// POST CloudEvents to the broker ingress, InMemoryChannels, JobSinks and
	// RequestReplies.
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-to-data-plane",
			Namespace: namespace,
			Labels:    managedLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: componentSelector("broker-ingress", "imc-dispatcher", "job-sink", "request-reply"),
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{}},
				Ports: networkPolicyPorts(
					networkPolicyPortTCP(dataPlaneHTTPPort),
					networkPolicyPortTCP(dataPlaneHTTPSPort),
				),
			}},
		},
	})

	// Event delivery: the data-plane roles that dial subscriber, sink and
	// target URIs in arbitrary user namespaces on arbitrary ports.
	policies = append(policies, networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-event-delivery",
			Namespace: namespace,
			Labels:    managedLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: componentSelector("broker-filter", "imc-dispatcher", "pingsource-mt-adapter", "request-reply"),
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: []networkingv1.NetworkPolicyPeer{{
					IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
				}},
				// No port scoping: subscriber URIs are arbitrary.
			}},
		},
	})

	return policies
}

// componentSelector selects the pods of the given eventing components via
// the app.kubernetes.io/component label.
func componentSelector(components ...string) metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      componentLabel,
			Operator: metav1.LabelSelectorOpIn,
			Values:   components,
		}},
	}
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
