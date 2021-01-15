package ingress

import (
	"context"
	"testing"
	"time"

	fakerouteclient "github.com/openshift-knative/serverless-operator/serving/ingress/pkg/client/injection/client/fake"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientgotesting "k8s.io/client-go/testing"
	"knative.dev/networking/pkg/apis/networking"
	"knative.dev/networking/pkg/apis/networking/v1alpha1"
	networkingclient "knative.dev/networking/pkg/client/injection/client/fake"
	ingressreconciler "knative.dev/networking/pkg/client/injection/reconciler/networking/v1alpha1/ingress"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/ptr"
	"knative.dev/serving/pkg/apis/serving"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress/resources"
	. "github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/testing"
	. "knative.dev/pkg/reconciler/testing"
)

const (
	ingName      = "test"
	ingNamespace = "testNs"
	ingUID       = "8a7e9a9d-fbc6-11e9-a88e-0261aff8d6d8"

	ingressNamespace = "knative-serving-ingress"

	svcName    = "kourier-ingressgateway"
	domainName = ingName + "." + ingNamespace + ".default.domainName"
	routeName  = "route-" + ingUID + "-306330363338"
)

func TestReconcile(t *testing.T) {
	key := ingNamespace + "/" + ingName

	table := TableTest{{
		Name: "bad workqueue key",
		Key:  "too/many/parts",
	}, {
		Name: "key not found",
		Key:  "foo/not-found",
	}, {
		Name:    "steady state",
		Key:     key,
		Objects: []runtime.Object{ing(ingNamespace, ingName), route(ingressNamespace, routeName)},
	}, {
		Name:                    "create route",
		SkipNamespaceValidation: true,
		Key:                     key,
		Objects:                 []runtime.Object{ing(ingNamespace, ingName)},
		WantCreates:             []runtime.Object{route(ingressNamespace, routeName)},
	}, {
		Name:                    "remove outdated routes",
		SkipNamespaceValidation: true,
		Key:                     key,
		Objects: []runtime.Object{
			ing(ingNamespace, ingName),
			route(ingressNamespace, routeName),
			route(ingressNamespace, "foo"), // This gets deleted.
			route(ingressNamespace, "foo2", func(r *routev1.Route) {
				r.Labels[resources.OpenShiftIngressLabelKey] = "foo"
				r.Labels[serving.RouteLabelKey] = "foo"
			}), // This doesn't cause the label doesn't match.
		},
		WantDeletes: []clientgotesting.DeleteActionImpl{{
			ActionImpl: clientgotesting.ActionImpl{
				Namespace: ingressNamespace,
				Resource:  routev1.SchemeGroupVersion.WithResource("routes"),
			},
			Name: "foo",
		}},
	}, {
		Name:                    "copy annotations and labels",
		SkipNamespaceValidation: true,
		Key:                     key,
		Objects: []runtime.Object{
			ing(ingNamespace, ingName, func(i *v1alpha1.Ingress) {
				i.Annotations["foo.bar/baz"] = "baz"
				i.Labels["foo.bar/baz"] = "baz"
			}),
		},
		WantCreates: []runtime.Object{
			route(ingressNamespace, routeName, func(r *routev1.Route) {
				r.Annotations["foo.bar/baz"] = "baz"
				r.Labels["foo.bar/baz"] = "baz"
			}),
		},
	}, {
		Name:                    "copy annotations and labels on update too",
		SkipNamespaceValidation: true,
		Key:                     key,
		Objects: []runtime.Object{
			ing(ingNamespace, ingName, func(i *v1alpha1.Ingress) {
				i.Annotations["foo.bar/baz"] = "baz"
				i.Labels["foo.bar/baz"] = "baz"
			}),
			route(ingressNamespace, routeName),
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: route(ingressNamespace, routeName, func(r *routev1.Route) {
				r.Annotations["foo.bar/baz"] = "baz"
				r.Labels["foo.bar/baz"] = "baz"
			}),
		}},
	}, {
		Name:                    "fix spec",
		SkipNamespaceValidation: true,
		Key:                     key,
		Objects: []runtime.Object{
			ing(ingNamespace, ingName),
			route(ingressNamespace, routeName, func(r *routev1.Route) {
				r.Spec.To.Kind = "foo"
			}),
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: route(ingressNamespace, routeName),
		}},
	}, {
		Name:                    "create nothing",
		SkipNamespaceValidation: true,
		Key:                     key,
		Objects: []runtime.Object{
			ing(ingNamespace, ingName, func(i *v1alpha1.Ingress) {
				i.Annotations[resources.DisableRouteAnnotation] = "true"
			}),
		},
	}, {
		Name:                    "add finalizer",
		SkipNamespaceValidation: true,
		Key:                     key,
		Objects: []runtime.Object{
			ing(ingNamespace, ingName, func(i *v1alpha1.Ingress) {
				i.Finalizers = []string{}
			}),
			route(ingressNamespace, routeName),
		},
		WantPatches: []clientgotesting.PatchActionImpl{
			{
				Name:       ingName,
				ActionImpl: clientgotesting.ActionImpl{Namespace: ingNamespace},
				Patch:      []byte(`{"metadata":{"finalizers":["ocp-ingress"],"resourceVersion":""}}`),
			},
		},
		WantEvents: []string{
			Eventf(corev1.EventTypeNormal, "FinalizerUpdate", "Updated %q finalizers", ingName),
		},
	}, {
		Name:                    "remove route and finalizer",
		SkipNamespaceValidation: true,
		Key:                     key,
		Objects: []runtime.Object{
			ing(ingNamespace, ingName, func(i *v1alpha1.Ingress) {
				i.DeletionTimestamp = &metav1.Time{
					Time: time.Now(),
				}
			}),
			route(ingressNamespace, routeName),
		},
		WantDeletes: []clientgotesting.DeleteActionImpl{{
			ActionImpl: clientgotesting.ActionImpl{
				Namespace: ingressNamespace,
				Resource:  routev1.SchemeGroupVersion.WithResource("routes"),
			},
			Name: routeName,
		}},
		WantPatches: []clientgotesting.PatchActionImpl{
			{
				Name:       ingName,
				ActionImpl: clientgotesting.ActionImpl{Namespace: ingNamespace},
				Patch:      []byte(`{"metadata":{"finalizers":[],"resourceVersion":""}}`),
			},
		},
		WantEvents: []string{
			Eventf(corev1.EventTypeNormal, "FinalizerUpdate", "Updated %q finalizers", ingName),
		},
	}, {
		// The new downstream label (OpenShiftIngressLabelKey) was introduced but Routes with old labels still should be reconciled.
		Name:                    "reconcile only old labels",
		SkipNamespaceValidation: true,
		Key:                     key,
		Objects: []runtime.Object{
			ing(ingNamespace, ingName),
			route(ingressNamespace, routeName, func(r *routev1.Route) {
				delete(r.Labels, resources.OpenShiftIngressLabelKey)
				delete(r.Labels, resources.OpenShiftIngressNamespaceLabelKey)
			}), // Test without downstream labels.
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: route(ingressNamespace, routeName),
		}},
	}}

	table.Test(t, MakeFactory(func(ctx context.Context, listers *Listers, cmw configmap.Watcher) controller.Reconciler {
		r := &Reconciler{
			routeClient: fakerouteclient.Get(ctx).RouteV1(),
			routeLister: listers.GetRouteLister(),
		}

		ingr := ingressreconciler.NewReconciler(ctx, logging.FromContext(ctx), networkingclient.Get(ctx),
			listers.GetIngressLister(), controller.GetEventRecorder(ctx), r, kourierIngressClassName,
			controller.Options{
				SkipStatusUpdates: true,
				FinalizerName:     "ocp-ingress",
			})

		return ingr
	}))
}

type ingressOption func(*v1alpha1.Ingress)

func ing(ns, name string, opts ...ingressOption) *v1alpha1.Ingress {
	i := &v1alpha1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			UID:         ingUID,
			Labels:      map[string]string{serving.RouteNamespaceLabelKey: ns, serving.RouteLabelKey: name},
			Annotations: map[string]string{networking.IngressClassAnnotationKey: kourierIngressClassName},
			Finalizers:  []string{"ocp-ingress"},
		},
		Spec: v1alpha1.IngressSpec{
			Rules: []v1alpha1.IngressRule{{
				Hosts: []string{domainName},
				HTTP: &v1alpha1.HTTPIngressRuleValue{
					Paths: []v1alpha1.HTTPIngressPath{{
						DeprecatedTimeout: &metav1.Duration{Duration: 5 * time.Second},
					}},
				},
			}},
		},
		Status: v1alpha1.IngressStatus{
			PublicLoadBalancer: &v1alpha1.LoadBalancerStatus{
				Ingress: []v1alpha1.LoadBalancerIngressStatus{{
					DomainInternal: svcName + "." + ingressNamespace + ".svc.cluster.local",
				}},
			},
		},
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

type routeOption func(*routev1.Route)

func route(ns, name string, opts ...routeOption) *routev1.Route {
	r := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				networking.IngressLabelKey:                  "test",
				serving.RouteLabelKey:                       "test",
				serving.RouteNamespaceLabelKey:              "testNs",
				resources.OpenShiftIngressLabelKey:          "test",
				resources.OpenShiftIngressNamespaceLabelKey: "testNs",
			},
			Annotations: map[string]string{
				resources.TimeoutAnnotation:          "5s",
				networking.IngressClassAnnotationKey: "kourier.ingress.networking.knative.dev",
			},
		},
		Spec: routev1.RouteSpec{
			Host: domainName,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http2"),
			},
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   svcName,
				Weight: ptr.Int32(100),
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyAllow,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}
