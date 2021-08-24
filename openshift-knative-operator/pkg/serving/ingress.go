package serving

import "knative.dev/operator/pkg/apis/operator/v1alpha1"

const istioIngressClassName = "istio.ingress.networking.knative.dev"

// defaultToKourier applies an Ingress config with Kourier enabled if nothing else is defined.
// Also handles the (buggy) case, where all Ingresses are disabled.
// See https://github.com/knative/operator/issues/568.
func defaultToKourier(ks *v1alpha1.KnativeServing) {
	if ks.Spec.Ingress == nil {
		ks.Spec.Ingress = &v1alpha1.IngressConfigs{
			Kourier: v1alpha1.KourierIngressConfiguration{
				Enabled: true,
			},
		}
		return
	}

	if !ks.Spec.Ingress.Istio.Enabled && !ks.Spec.Ingress.Kourier.Enabled && !ks.Spec.Ingress.Contour.Enabled {
		ks.Spec.Ingress.Kourier.Enabled = true
	}
}

// defaultIngressClass tries to figure out which ingress class to default to.
// - If nothing is defined, Kourier will be used.
// - If Kourier is enabled, it'll always take precedence.
// - If only Istio is enabled, it'll be used.
func defaultIngressClass(ks *v1alpha1.KnativeServing) string {
	if ks.Spec.Ingress == nil {
		return kourierIngressClassName
	}
	if ks.Spec.Ingress.Kourier.Enabled {
		return kourierIngressClassName
	}
	if ks.Spec.Ingress.Istio.Enabled {
		return istioIngressClassName
	}
	return kourierIngressClassName
}
