package serving

import (
	"strconv"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

const (
	istioIngressClassName                       = "istio.ingress.networking.knative.dev"
	disableGeneratingIstioNetPoliciesAnnotation = "serverless.openshift.io/disable-istio-net-policies-generation"
)

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

func generateDefaultIstioNetworkPoliciesIfRequired(ks base.KComponent) ([]mf.Manifest, error) {
	if !ks.(*operatorv1beta1.KnativeServing).Spec.Ingress.Istio.Enabled {
		return nil, nil
	} else {
		if v, ok := ks.GetAnnotations()[disableGeneratingIstioNetPoliciesAnnotation]; ok {
			b, _ := strconv.ParseBool(v)
			if b {
				return nil, nil
			}
		}
	}
	unObjs := []unstructured.Unstructured{unstructured.Unstructured{}, unstructured.Unstructured{}, unstructured.Unstructured{}}
	for i, name := range []string{"webhook", "net-istio-webhook", "allow-from-openshift-monitoring-ns"} {
		nwp := v1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ks.GetNamespace(),
				Labels: map[string]string{
					"networking.knative.dev/ingress-provider": "istio",
				},
			},
		}
		if name != "allow-from-openshift-monitoring-ns" {
			nwp.Labels["app"] = name
			nwp.Spec.PodSelector = metav1.LabelSelector{MatchLabels: map[string]string{
				"app": name,
			}}
			nwp.Spec.Ingress = []v1.NetworkPolicyIngressRule{{}}
		} else {
			nwp.Spec.PodSelector = metav1.LabelSelector{}
			nwp.Spec.PolicyTypes = []v1.PolicyType{v1.PolicyTypeIngress}
			nwp.Spec.Ingress = []v1.NetworkPolicyIngressRule{v1.NetworkPolicyIngressRule{
				From: []v1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": monitoring.OpenshiftMonitoringNamespace,
						},
					},
				},
				},
			}}
		}
		if err := scheme.Scheme.Convert(&nwp, &unObjs[i], nil); err != nil {
			return nil, err
		}
	}
	m, err := mf.ManifestFrom(mf.Slice(unObjs))
	if err != nil {
		return nil, err
	} else {
		return []mf.Manifest{m}, nil
	}
}
