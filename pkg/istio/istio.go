package istio

import (
	"context"

	mf "github.com/manifestival/manifestival"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

func GetServiceMeshNetworkPolicy() (mf.Manifest, error) {
	networkPolicies := serviceMeshNetworkPolicies()
	networkPoliciesUnstr, err := toUnstructured(networkPolicies)
	if err != nil {
		return mf.Manifest{}, err
	}

	m, err := mf.ManifestFrom(mf.Slice(networkPoliciesUnstr))
	if err != nil {
		return m, err
	}
	return m, nil
}

func IsEnabled(k kubernetes.Interface) (bool, error) {
	ns, err := k.CoreV1().Namespaces().Get(context.Background(), "knative-eventing", metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	v, ok := ns.Labels["maistra.io/member-of"]
	if ok && v != "" {
		return true, nil
	}
	return false, nil
}

func toUnstructured(policies []networkingv1.NetworkPolicy) ([]unstructured.Unstructured, error) {
	r := make([]unstructured.Unstructured, 0, len(policies))
	for _, p := range policies {
		p := p
		u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&p)
		if err != nil {
			return nil, err
		}
		r = append(r, unstructured.Unstructured{Object: u})
	}
	return r, nil
}

func serviceMeshNetworkPolicies() []networkingv1.NetworkPolicy {

	gvk := networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy")

	tm := metav1.TypeMeta{
		Kind:       gvk.Kind,
		APIVersion: gvk.GroupVersion().String(),
	}

	return []networkingv1.NetworkPolicy{
		{
			TypeMeta: tm,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allow-from-openshift-monitoring",
				Namespace: "knative-eventing",
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{
						From: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: nil,
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"kubernetes.io/metadata.name": "openshift-monitoring",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			TypeMeta: tm,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allow-eventing-webhook",
				Namespace: "knative-eventing",
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app.kubernetes.io/component": "eventing-webhook",
					},
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{{}},
			},
		},
		{
			TypeMeta: tm,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allow-eventing-webhook",
				Namespace: "knative-eventing",
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app.kubernetes.io/component": "imc-controller",
					},
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{{}},
			},
		},
		{
			TypeMeta: tm,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kafka-webhook-eventing",
				Namespace: "knative-eventing",
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "kafka-webhook-eventing",
					},
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{{}},
			},
		},
	}
}
