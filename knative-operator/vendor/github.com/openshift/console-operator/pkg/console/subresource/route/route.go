package route

import (
	// kube
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"

	operatorv1 "github.com/openshift/api/operator/v1"
	routev1 "github.com/openshift/api/route/v1"
	routeclient "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	"github.com/openshift/console-operator/pkg/console/subresource/util"
)

const (
	// ingress instance named "default" is the OOTB ingresscontroller
	// this is an implicit stable API
	defaultIngressController = "default"
)

// ensures route exists.
// handles 404 with a create
// returns any other error
func GetOrCreate(client routeclient.RoutesGetter, required *routev1.Route) (*routev1.Route, bool, error) {
	isNew := false
	route, err := client.Routes(required.Namespace).Get(required.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		isNew = true
		route, err = client.Routes(required.Namespace).Create(required)
	}
	if err != nil {
		return nil, isNew, err
	}
	return route, isNew, nil
}

func DefaultRoute(cr *operatorv1.Console) *routev1.Route {
	route := Stub()
	route.Spec = routev1.RouteSpec{
		To:             toService(),
		Port:           port(),
		TLS:            tls(),
		WildcardPolicy: wildcard(),
	}
	util.AddOwnerRef(route, util.OwnerRefFrom(cr))
	return route
}

func Stub() *routev1.Route {
	meta := util.SharedMeta()
	return &routev1.Route{
		ObjectMeta: meta,
	}
}

// we can't just blindly apply the route, we need the route.Spec.Host
// and we don't want to trigger a sync loop.
// TODO: evaluate metadata.annotations to see what will affect our route in
// an undesirable way:
// - https://docs.openshift.com/container-platform/3.9/architecture/networking/routes.html#alternateBackends
func Validate(route *routev1.Route) (*routev1.Route, bool) {
	changed := false

	if toServiceSame := equality.Semantic.DeepEqual(route.Spec.To, toService()); !toServiceSame {
		changed = true
		route.Spec.To = toService()
	}

	if portSame := equality.Semantic.DeepEqual(route.Spec.Port, port()); !portSame {
		changed = true
		route.Spec.Port = port()
	}

	if tlsSame := equality.Semantic.DeepEqual(route.Spec.TLS, tls()); !tlsSame {
		changed = true
		route.Spec.TLS = tls()
	}

	if wildcardSame := equality.Semantic.DeepEqual(route.Spec.WildcardPolicy, wildcard()); !wildcardSame {
		changed = true
		route.Spec.WildcardPolicy = wildcard()
	}

	return route, changed
}

func routeMeta() metav1.ObjectMeta {
	meta := util.SharedMeta()
	return meta
}

func toService() routev1.RouteTargetReference {
	weight := int32(100)
	return routev1.RouteTargetReference{
		Kind:   "Service",
		Name:   routeMeta().Name,
		Weight: &weight,
	}
}

func port() *routev1.RoutePort {
	return &routev1.RoutePort{
		TargetPort: intstr.FromString("https"),
	}
}

func tls() *routev1.TLSConfig {
	return &routev1.TLSConfig{
		Termination:                   routev1.TLSTerminationReencrypt,
		InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
	}
}

func wildcard() routev1.WildcardPolicyType {
	return routev1.WildcardPolicyNone
}

func GetCanonicalHost(route *routev1.Route) string {
	for _, ingress := range route.Status.Ingress {
		if ingress.RouterName != defaultIngressController {
			klog.V(4).Infof("ignoring route ingress '%v'", ingress.RouterName)
			continue
		}
		// ingress must be admitted before it is useful to us
		if !isIngressAdmitted(ingress) {
			klog.V(4).Infof("route ingress '%v' not admitted", ingress.RouterName)
			continue
		}
		klog.V(4).Infof("route ingress '%v' found and admitted, host: %v", defaultIngressController, ingress.Host)
		return ingress.Host
	}
	klog.V(4).Infoln("route ingress not yet ready for console")
	return ""
}

// for the purpose of availability, we simply need to know when the
// route has been admitted.  we may have multiple ingress on the route, each
// with an admitted attribute.
func IsAdmitted(route *routev1.Route) bool {
	for _, ingress := range route.Status.Ingress {
		if isIngressAdmitted(ingress) {
			return true
		}
	}
	return false
}

func isIngressAdmitted(ingress routev1.RouteIngress) bool {
	admitted := false
	for _, condition := range ingress.Conditions {
		if condition.Type == routev1.RouteAdmitted && condition.Status == corev1.ConditionTrue {
			admitted = true
		}
	}
	return admitted
}
