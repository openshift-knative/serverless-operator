package resources

import (
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/pkg/ptr"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
)

func MakeSkupperResources(i *networkingv1alpha1.Ingress) (*corev1.Service, []*routev1.Route) {
	svcName := i.Name + "-skupper"
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      svcName,
			Annotations: map[string]string{
				"skupper.io/proxy":  "http",
				"skupper.io/target": i.Name + "." + i.Namespace,
			},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(i, networkingv1alpha1.SchemeGroupVersion.WithKind("Ingress"))},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name: "http",
				Port: 80,
			}},
		},
	}

	routes := make([]*routev1.Route, 0, len(i.Spec.Rules))
	for _, rule := range i.Spec.Rules {
		// Skip making route when visibility of the rule is local only.
		if rule.Visibility == networkingv1alpha1.IngressVisibilityClusterLocal {
			continue
		}

		for _, host := range rule.Hosts {
			route := &routev1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       i.Namespace,
					Name:            routeName(string(i.UID), host),
					OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(i, networkingv1alpha1.SchemeGroupVersion.WithKind("Ingress"))},
				},
				Spec: routev1.RouteSpec{
					Host: host,
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString("http"),
					},
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   svcName,
						Weight: ptr.Int32(100),
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
				},
			}
			routes = append(routes, route)
		}
	}

	return svc, routes
}
