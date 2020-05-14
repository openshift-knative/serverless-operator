package resources

import (
	//	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/resources"
)

// MakePublicEndpoints constructs a K8s Endpoints that is not backed a selector
// and will be manually reconciled by the SKS controller.
func MakeEndpoints(src *corev1.Endpoints, ing *networkingv1alpha1.Ingress, name string) *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name, // Name of Endpoints must match that of Service.
			Namespace:   ing.Namespace,
			Labels:      resources.UnionMaps(ing.GetLabels(), map[string]string{}),
			Annotations: resources.CopyMap(ing.GetAnnotations()),
		},
		Subsets: src.Subsets,
	}
}
