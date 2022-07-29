package resources

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/networking/pkg/apis/networking"
	networkingv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	"knative.dev/pkg/kmap"
	"knative.dev/pkg/ptr"
	"knative.dev/serving/pkg/apis/config"
)

const (
	TimeoutAnnotation                = "haproxy.router.openshift.io/timeout"
	DisableRouteAnnotation           = socommon.ServingDownstreamDomain + "/disableRoute"
	EnablePassthroughRouteAnnotation = socommon.ServingDownstreamDomain + "/enablePassthrough"

	HTTPPort  = "http2"
	HTTPSPort = "https"

	OpenShiftIngressLabelKey          = socommon.ServingDownstreamDomain + "/ingressName"
	OpenShiftIngressNamespaceLabelKey = socommon.ServingDownstreamDomain + "/ingressNamespace"
)

// DefaultTimeout is set by DefaultMaxRevisionTimeoutSeconds. So, the OpenShift Route's timeout
// should not have any effect on Knative services by default.
var DefaultTimeout = fmt.Sprintf("%vs", config.DefaultMaxRevisionTimeoutSeconds)

// ErrNoValidLoadbalancerDomain indicates that the current ingress does not have a DomainInternal field, or
// said field does not contain a value we can work with.
var ErrNoValidLoadbalancerDomain = errors.New("unable to find Ingress LoadBalancer with DomainInternal set")

// MakeRoutes creates OpenShift Routes from a Knative Ingress
func MakeRoutes(ci *networkingv1alpha1.Ingress) ([]*routev1.Route, error) {
	routes := []*routev1.Route{}

	for _, rule := range ci.Spec.Rules {
		// Skip route creation for cluster-local visibility.
		if rule.Visibility == networkingv1alpha1.IngressVisibilityClusterLocal {
			continue
		}
		for _, host := range rule.Hosts {
			// Ignore domains like myksvc.myproject.svc.cluster.local
			parts := strings.Split(host, ".")
			if len(parts) == 2 || (len(parts) > 2 && parts[2] != "svc") {
				route, err := makeRoute(ci, host, rule)
				if err != nil {
					return nil, err
				}
				if route == nil {
					continue
				}
				routes = append(routes, route)
			}
		}
	}

	return routes, nil
}

func makeRoute(ci *networkingv1alpha1.Ingress, host string, rule networkingv1alpha1.IngressRule) (*routev1.Route, error) {
	// Take over annotaitons from ingress.
	annotations := ci.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Skip making route when visibility of the rule is local only.
	if rule.Visibility == networkingv1alpha1.IngressVisibilityClusterLocal {
		return nil, nil
	}

	// Skip making route when the annotation is specified.
	if _, ok := annotations[DisableRouteAnnotation]; ok {
		return nil, nil
	}

	// Set timeout for OpenShift Route
	annotations[TimeoutAnnotation] = DefaultTimeout

	labels := kmap.Union(ci.Labels, map[string]string{
		networking.IngressLabelKey:        ci.GetName(),
		OpenShiftIngressLabelKey:          ci.GetName(),
		OpenShiftIngressNamespaceLabelKey: ci.GetNamespace(),
	})

	name := routeName(string(ci.GetUID()), host)
	serviceName := ""
	namespace := ""
	if ci.Status.PublicLoadBalancer != nil {
		for _, lbIngress := range ci.Status.PublicLoadBalancer.Ingress {
			if lbIngress.DomainInternal != "" {
				// DomainInternal should look something like:
				// kourier.knative-serving-ingress.svc.cluster.local
				parts := strings.Split(lbIngress.DomainInternal, ".")
				if len(parts) > 2 && parts[2] == "svc" {
					serviceName = parts[0]
					namespace = parts[1]
				}
			}
		}
	}

	if serviceName == "" || namespace == "" {
		return nil, ErrNoValidLoadbalancerDomain
	}

	terminationPolicy := routev1.InsecureEdgeTerminationPolicyAllow
	if ci.Spec.HTTPOption == networkingv1alpha1.HTTPOptionRedirected {
		terminationPolicy = routev1.InsecureEdgeTerminationPolicyRedirect
	}

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: routev1.RouteSpec{
			Host: host,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(HTTPPort),
			},
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   serviceName,
				Weight: ptr.Int32(100),
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: terminationPolicy,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}

	// Use passthrough type for OpenShift Ingress -> Kourier Gateway to encrypt the traffic
	// when Internal TLS is enabled.
	// In other words, do not use edge termination which makes plain traffic.
	if isInternalEncryptionEnabled(rule) {
		route.Spec.Port = &routev1.RoutePort{
			TargetPort: intstr.FromString(HTTPSPort),
		}
		route.Spec.TLS = &routev1.TLSConfig{
			Termination:                   routev1.TLSTerminationPassthrough,
			InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
		}
	}

	// Target the HTTPS port and configure passthrough when:
	// * the passthrough annotation is set.
	// * the ingress.spec.tls is set. (DomainMapping with BYP cert.)
	if _, ok := annotations[EnablePassthroughRouteAnnotation]; ok || len(ci.Spec.TLS) > 0 {
		route.Spec.Port.TargetPort = intstr.FromString(HTTPSPort)
		route.Spec.TLS.Termination = routev1.TLSTerminationPassthrough
		route.Spec.TLS.InsecureEdgeTerminationPolicy = routev1.InsecureEdgeTerminationPolicyRedirect
	}

	return route, nil
}

// isInternalEncryptionEnabled determines whether internal-encryption is enabled or not.
// In general, we can determine it by the value internal-encryption in config-network, however the serverless ingress does not
// watch the ConfigMap. Therefore we determine it by ServiceHTTPSPort(443) port for the backend in Kingress.
func isInternalEncryptionEnabled(rule networkingv1alpha1.IngressRule) bool {
	if rule.HTTP == nil {
		return false
	}
	for _, path := range rule.HTTP.Paths {
		for _, split := range path.Splits {
			if split.IngressBackend.ServicePort == intstr.FromInt(networking.ServiceHTTPSPort) {
				return true
			}
		}
	}
	return false
}

func routeName(uid, host string) string {
	return fmt.Sprintf("route-%s-%x", uid, hashHost(host))
}

func hashHost(host string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(host)))[0:6]
}
