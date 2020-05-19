package resources

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/resources"
)

// MakeK8sService constructs a K8s Service to expose Kourier gateway endpoints.
func MakeK8sService(ctx context.Context, ing *networkingv1alpha1.Ingress, name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ing.Namespace,
			Labels:      resources.UnionMaps(ing.GetLabels(), map[string]string{}),
			Annotations: resources.CopyMap(ing.GetAnnotations()),
		},

		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name: KourierHttpPort,
				Port: 80,
			}},
		},
	}
}
