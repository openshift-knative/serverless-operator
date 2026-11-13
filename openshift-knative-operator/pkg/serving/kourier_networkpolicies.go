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
	"net"
	"strconv"
	"strings"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/networkpolicy"
)

const (
	// kourierGatewayPodLabel selects the Envoy gateway deployment.
	kourierGatewayPodLabel = "3scale-kourier-gateway"

	// kourierGatewayHTTPPort, kourierGatewayInternalPort,
	// kourierGatewayHTTPSPort and kourierGatewayInternalTLSPort are the
	// gateway service targetPorts: external and cluster-local traffic, plus
	// their TLS counterparts (https-external, kourier-internal https).
	kourierGatewayHTTPPort        int32 = 8080
	kourierGatewayInternalPort    int32 = 8081
	kourierGatewayHTTPSPort       int32 = 8443
	kourierGatewayInternalTLSPort int32 = 8444

	// monitoringPort is the kube-rbac-proxy secure metrics port, matching
	// the shared builder.
	monitoringPort int32 = 8444

	// unixMaxPort is the highest registered port number.
	unixMaxPort = 65535
)

// generateKourierNetworkPolicies returns the NetworkPolicies for the Kourier
// namespace (knative-serving-ingress), or nil when generation is disabled
// via the opt-out annotation or when Kourier is not the enabled ingress (the
// namespace only exists with Kourier).
//
// The policies carry the Kourier ingress-provider label, so the
// overrideKourierNamespace transformer moves them into the Kourier namespace
// and strips owner references along with the other Kourier resources.
//
// kubeclient is used to resolve the backend port of a configured external
// authorization service; it may be nil, in which case only the configured
// service port is allowed.
func generateKourierNetworkPolicies(ks base.KComponent, kubeclient kubernetes.Interface) ([]mf.Manifest, error) {
	if networkPoliciesDisabled(ks) || !kourierIngressEnabled(ks) {
		return nil, nil
	}
	extAuthz := parseExtAuthzEndpoint(ks)
	if extAuthz != nil && kubeclient != nil {
		if err := extAuthz.resolveBackendPort(kubeclient); err != nil {
			return nil, err
		}
	}
	return networkpolicy.ToManifests(kourierNetworkPolicies(kourierNamespace(ks.GetNamespace()), extAuthz))
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
// for the Envoy gateway data plane (see SRVCOM-3716). extAuthz, when
// non-empty, is the configured external authorization service
// (namespace, port) and yields an additional scoped gateway egress rule.
func kourierNetworkPolicies(namespace string, extAuthz *extAuthzEndpoint) []networkingv1.NetworkPolicy {
	// All policies carry the Kourier provider label so the
	// overrideKourierNamespace transformer moves them into the Kourier
	// namespace and strips owner references (a cross-namespace owner ref
	// would make the garbage collector delete them).
	policies := []networkingv1.NetworkPolicy{
		networkpolicy.DenyAll("deny-by-default", namespace),
		networkpolicy.AllowSameNamespace("allow-same-namespace", namespace),
		networkpolicy.AllowToAPIServer("allow-to-api-server", namespace),
		networkpolicy.AllowToDNS("allow-to-dns", namespace),
		kourierMonitoringPolicy(namespace),
		networkpolicy.AllowToTracing("allow-to-tracing", namespace),
	}
	for i := range policies {
		policies[i].Labels[providerLabel] = "kourier"
	}

	// The gateway receives external traffic from the OpenShift router and
	// cluster-local traffic from arbitrary user namespaces (ksvc URLs
	// resolve to kourier-internal), plus TLS traffic on https-external
	// (8443) and cluster-local TLS (8444). No From peers: all sources (an
	// empty peer would be rejected by the API server).
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
				Ports: networkPolicyPorts(
					networkPolicyPortTCP(kourierGatewayHTTPPort),
					networkPolicyPortTCP(kourierGatewayInternalPort),
					networkPolicyPortTCP(kourierGatewayHTTPSPort),
					networkPolicyPortTCP(kourierGatewayInternalTLSPort),
				),
			}},
		},
	})

	// Gateway -> activator and queue-proxies of revisions in arbitrary user
	// namespaces on the HTTP, h2c and HTTPS backend ports.
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
				To: networkpolicy.AnyAddressPeers(),
				Ports: networkPolicyPorts(
					networkPolicyPortTCP(activatorPort),
					networkPolicyPortTCP(activatorHTTP2Port),
					networkPolicyPortTCP(activatorHTTPSPort),
				),
			}},
		},
	})

	// Gateway -> configured external authorization service, scoped to its
	// namespace and ports (the service port, and the backend port when it
	// differs: OVN-Kubernetes applies egress policies after service load
	// balancing). Named backend ports are kept as names: NetworkPolicy
	// ports resolve them against the backend pods.
	if extAuthz != nil {
		ports := networkPolicyPorts(networkPolicyPortTCP(extAuthz.port))
		if extAuthz.backendPort != 0 && extAuthz.backendPort != extAuthz.port {
			ports = append(ports, networkPolicyPortTCP(extAuthz.backendPort))
		}
		if extAuthz.backendPortName != "" {
			ports = append(ports, networkingv1.NetworkPolicyPort{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromString(extAuthz.backendPortName)),
			})
		}
		policies = append(policies, networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allow-gateway-to-extauthz",
				Namespace: namespace,
				Labels:    kourierLabels(),
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": kourierGatewayPodLabel}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
				Egress: []networkingv1.NetworkPolicyEgressRule{{
					To: []networkingv1.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": extAuthz.namespace,
							},
						},
					}},
					Ports: ports,
				}},
			},
		})
	}

	return policies
}

// knativeServingsForExtAuthzService returns the keys of the KnativeServings
// whose configured external authorization service is the given Service. It
// backs the Service watch: changes to such a Service re-enqueue the owning
// KnativeServing, so the generated egress rule follows the Service's ports
// and creation.
func knativeServingsForExtAuthzService(knativeServings []*operatorv1beta1.KnativeServing, serviceNamespace, serviceName string) []types.NamespacedName {
	var keys []types.NamespacedName
	for _, ks := range knativeServings {
		if extAuthz := parseExtAuthzEndpoint(ks); extAuthz != nil &&
			extAuthz.namespace == serviceNamespace && extAuthz.service == serviceName {
			keys = append(keys, types.NamespacedName{Namespace: ks.Namespace, Name: ks.Name})
		}
	}
	return keys
}

// deleteExtAuthzNetworkPolicy removes a stale allow-gateway-to-extauthz
// policy from the Kourier namespace, used when the external authorization
// service is no longer configured: extension manifests are not pruned
// automatically, so the allow would otherwise linger. Only policies carrying
// the managed-by label are removed.
func deleteExtAuthzNetworkPolicy(ctx context.Context, kubeclient kubernetes.Interface, namespace string) error {
	p, err := kubeclient.NetworkingV1().NetworkPolicies(namespace).Get(ctx, "allow-gateway-to-extauthz", metav1.GetOptions{})
	if err != nil {
		return err
	}
	if p.Labels[networkpolicy.ManagedByLabel] != "true" {
		// Not ours: leave it alone.
		return nil
	}
	return kubeclient.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, p.Name, metav1.DeleteOptions{})
}

// extAuthzEndpoint describes a configured Kourier external authorization
// service (config-kourier extauthz-host) as far as a NetworkPolicy can
// express it: the namespace of the service, its name and the ports to allow.
type extAuthzEndpoint struct {
	namespace string
	// service is the authorization service's name, used to resolve the
	// backend (targetPort) port.
	service string
	// port is the configured service port.
	port int32
	// backendPort is the resolved numeric backend port, when it differs
	// from the service port. OVN-Kubernetes applies egress policies after
	// service load balancing (post-DNAT), so the backend port must be
	// allowed too.
	backendPort int32
	// backendPortName is the resolved named backend port (targetPort).
	// NetworkPolicy ports accept named ports, which resolve against the
	// backend pods, so the name can be used directly.
	backendPortName string
}

// parseExtAuthzEndpoint parses the extauthz-host from the Kourier
// configuration (via the KnativeServing spec). Both the config-kourier and
// the short kourier key are supported, matching the operator's ConfigMap
// merge precedence (the full name wins). It returns nil when no rule can be
// generated: no service configured, a bare service name (same namespace,
// covered by allow-same-namespace) or an external hostname that cannot be
// scoped by a namespace selector.
func parseExtAuthzEndpoint(ks base.KComponent) *extAuthzEndpoint {
	host := ""
	if cfg := ks.GetSpec().GetConfig(); cfg != nil {
		if kourierCfg, ok := cfg["config-kourier"]; ok {
			host = kourierCfg["extauthz-host"]
		} else if kourierCfg, ok := cfg["kourier"]; ok {
			// The "config-" prefix is optional in spec.config, see
			// common.ConfigMapTransform.
			host = kourierCfg["extauthz-host"]
		}
	}
	if host == "" {
		return nil
	}

	h, portStr, err := net.SplitHostPort(host)
	if err != nil {
		return nil
	}
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil || port <= 0 {
		return nil
	}

	// Only cluster service DNS names can be scoped: <service>.<namespace>.svc
	// optionally with further suffixes (svc.cluster.local and beyond).
	labels := strings.Split(h, ".")
	if len(labels) < 3 || labels[2] != "svc" {
		return nil
	}
	return &extAuthzEndpoint{namespace: labels[1], service: labels[0], port: int32(port)}
}

// resolveBackendPort looks up the authorization service and records its
// backend port (targetPort) when it differs from the service port: numeric
// targetPorts as backendPort, named targetPorts as backendPortName.
//
// A missing Service is not an error: the rule falls back to the service
// port and the Service watch re-enqueues the KnativeServing once the
// Service appears. Any other lookup error is returned so the reconcile
// retries instead of silently keeping an incomplete policy.
func (e *extAuthzEndpoint) resolveBackendPort(kubeclient kubernetes.Interface) error {
	svc, err := kubeclient.CoreV1().Services(e.namespace).Get(context.Background(), e.service, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	for _, p := range svc.Spec.Ports {
		if p.Port != e.port {
			continue
		}
		switch p.TargetPort.Type {
		case intstr.Int:
			if backend := p.TargetPort.IntValue(); backend > 0 && backend <= unixMaxPort {
				//nolint:gosec // bounded by unixMaxPort above
				if int32(backend) != e.port {
					e.backendPort = int32(backend)
				}
			}
		case intstr.String:
			e.backendPortName = p.TargetPort.StrVal
		}
	}
	return nil
}

// kourierMonitoringPolicy allows the cluster-monitoring prometheus to scrape
// the Kourier namespace pods. Unlike the serving/eventing components, the
// Kourier pods get no kube-rbac-proxy injected (see servingDeployments in
// pkg/monitoring), so their metrics are exposed directly on 9090 in addition
// to the common 8444.
func kourierMonitoringPolicy(namespace string) networkingv1.NetworkPolicy {
	return networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-from-openshift-monitoring",
			Namespace: namespace,
			Labels:    managedLabels(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "openshift-monitoring",
						},
					},
				}},
				Ports: networkPolicyPorts(
					networkPolicyPortTCP(monitoringPort),
					networkPolicyPortTCP(9090),
				),
			}},
		},
	}
}

func kourierLabels() map[string]string {
	labels := managedLabels()
	labels[providerLabel] = "kourier"
	return labels
}
