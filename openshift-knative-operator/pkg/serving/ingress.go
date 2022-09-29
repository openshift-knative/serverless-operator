package serving

import (
	corev1 "k8s.io/api/core/v1"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

const istioIngressClassName = "istio.ingress.networking.knative.dev"

// defaultToKourier applies an Ingress config with Kourier enabled if nothing else is defined.
// Also handles the (buggy) case, where all Ingresses are disabled.
// See https://github.com/knative/operator/issues/568.
func defaultToKourier(ks *operatorv1beta1.KnativeServing) {
	if ks.Spec.Ingress == nil {
		ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{
			Kourier: base.KourierIngressConfiguration{
				Enabled: true,
			},
		}
		return
	}

	if !ks.Spec.Ingress.Istio.Enabled && !ks.Spec.Ingress.Kourier.Enabled && !ks.Spec.Ingress.Contour.Enabled {
		ks.Spec.Ingress.Kourier.Enabled = true
	}

}

func defaultKourierServiceType(ks *operatorv1beta1.KnativeServing) {
	if ks.Spec.Ingress != nil && ks.Spec.Ingress.Kourier.Enabled {
		if ks.Spec.Ingress.Kourier.ServiceType == "" {
			ks.Spec.Ingress.Kourier.ServiceType = corev1.ServiceTypeClusterIP
		}
	}
}

// defaultIngressClass tries to figure out which ingress class to default to.
// - If nothing is defined, Kourier will be used.
// - If Kourier is enabled, it'll always take precedence.
// - If only Istio is enabled, it'll be used.
func defaultIngressClass(ks *operatorv1beta1.KnativeServing) string {
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
