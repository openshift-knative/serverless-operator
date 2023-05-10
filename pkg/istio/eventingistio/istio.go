package eventingistio

import (
	"context"
	"fmt"
	"strings"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func IsEnabled(data base.ConfigMapData) bool {
	featuresConfigMap := getFeaturesConfig(data)
	v, ok := featuresConfigMap["istio"]
	return ok && strings.EqualFold(v, "enabled")
}

func getFeaturesConfig(cfg base.ConfigMapData) map[string]string {
	if v, ok := cfg["features"]; ok {
		return v
	}
	if v, ok := cfg["config-features"]; ok {
		return v
	}
	return nil
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
				Name:      "allow-imc-webhook",
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
				Name:      "allow-kafka-webhook-eventing",
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

func MaybeScaleIstioController(client client.Client, eventing *operatorv1beta1.KnativeEventing) error {
	if enabled := IsEnabled(eventing.GetSpec().GetConfig()); enabled {
		return scaleEventingIstioController(client, eventing, func(d *appsv1.Deployment) (bool, int32) {
			return *d.Spec.Replicas < 1, 1
		})
	}
	return scaleEventingIstioController(client, eventing, func(d *appsv1.Deployment) (bool, int32) {
		return *d.Spec.Replicas > 0, 0
	})
}

func scaleEventingIstioController(client client.Client, eventing *operatorv1beta1.KnativeEventing, shouldScaleFn func(d *appsv1.Deployment) (bool, int32)) error {
	istioControllerName := types.NamespacedName{Namespace: eventing.GetNamespace(), Name: "eventing-istio-controller"}

	overrides := eventing.GetSpec().GetWorkloadOverrides()
	for _, v := range overrides {
		if v.Name == istioControllerName.Name {
			if v.Replicas != nil {
				return nil
			}
		}
	}

	istioController := &appsv1.Deployment{}
	if err := client.Get(context.Background(), istioControllerName, istioController); err != nil {
		return fmt.Errorf("failed to get %s: %w", istioControllerName.String(), err)
	}

	if shouldScale, replicas := shouldScaleFn(istioController); shouldScale {
		istioController = istioController.DeepCopy()
		istioController.Spec.Replicas = pointer.Int32(replicas)
		if err := client.Update(context.Background(), istioController); err != nil {
			return fmt.Errorf("failed to update %s: %w", istioControllerName, err)
		}
	}

	return nil
}
