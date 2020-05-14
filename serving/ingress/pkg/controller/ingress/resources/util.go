package resources

import (
	"errors"
	"strings"

	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
)

// ErrNoValidLoadbalancerDomain indicates that the current ingress does not have a DomainInternal field, or
// said field does not contain a value we can work with.
var ErrNoValidLoadbalancerDomain = errors.New("unable to find Ingress LoadBalancer with DomainInternal set")

func IngressName(ing *networkingv1alpha1.Ingress) (string, string, error) {
	serviceName := ""
	namespace := ""
	if ing.Status.LoadBalancer != nil {
		for _, lbIngress := range ing.Status.LoadBalancer.Ingress {
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
		return "", "", ErrNoValidLoadbalancerDomain
	}
	return serviceName, namespace, nil
}
