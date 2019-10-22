package knativeserving

import (
	"context"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// configure egress
func (a *KnativeServingConfigurator) egress(ctx context.Context, ks *servingv1alpha1.KnativeServing) error {
	networkConfig := &configv1.Network{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: "cluster"}, networkConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return err
		}
		log.Info("No OpenShift cluster network config available")
		return nil
	}
	network := strings.Join(networkConfig.Spec.ServiceNetwork, ",")
	return configure(ks, "network", "istio.sidecar.includeOutboundIPRanges", network)
}

// configure ingress
func (a *KnativeServingConfigurator) ingress(ctx context.Context, ks *servingv1alpha1.KnativeServing) error {
	ingressConfig := &configv1.Ingress{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return err
		}
		log.Info("No OpenShift ingress config available")
		return nil
	}
	domain := ingressConfig.Spec.Domain
	if len(domain) > 0 {
		return configure(ks, "domain", domain, "")
	}
	return nil
}
