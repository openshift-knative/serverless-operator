package ingress

import (
	"context"
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/controller/ingress/resources"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/serving/pkg/apis/networking"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/apis/serving"
	"knative.dev/serving/pkg/network"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	name                 = "ingress-operator"
	serviceMeshNamespace = "knative-serving-ingress"
	namespace            = "ingress-namespace"
	uid                  = "8a7e9a9d-fbc6-11e9-a88e-0261aff8d6d8"
	domainName           = name + "." + namespace + ".default.domainName"
	routeName0           = "route-" + uid + "-336636653035"
)

var (
	defaultIngress = &networkingv1alpha1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			UID:         uid,
			Labels:      map[string]string{serving.RouteNamespaceLabelKey: namespace, serving.RouteLabelKey: name},
			Annotations: map[string]string{networking.IngressClassAnnotationKey: network.IstioIngressClassName},
		},
		Spec: networkingv1alpha1.IngressSpec{
			Visibility: networkingv1alpha1.IngressVisibilityExternalIP,
			Rules: []networkingv1alpha1.IngressRule{{
				Hosts: []string{domainName},
				HTTP: &networkingv1alpha1.HTTPIngressRuleValue{
					Paths: []networkingv1alpha1.HTTPIngressPath{{
						Timeout: &metav1.Duration{Duration: 5 * time.Second},
					}},
				},
			}},
		},
		Status: networkingv1alpha1.IngressStatus{
			LoadBalancer: &networkingv1alpha1.LoadBalancerStatus{
				Ingress: []networkingv1alpha1.LoadBalancerIngressStatus{{
					DomainInternal: "istio-ingressgateway." + serviceMeshNamespace + ".svc.cluster.local",
				}},
			},
		},
	}
)

func TestRouteMigration(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	var (
		noRemoveOtherLabel = routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "no-remove-other-label",
				Labels: map[string]string{networking.IngressLabelKey: "another", serving.RouteLabelKey: name, serving.RouteNamespaceLabelKey: namespace},
			},
			Spec: routev1.RouteSpec{Host: "c.example.com"},
		}
		noRemoveMissingLabel = routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "no-remove-missing-label",
				Labels: map[string]string{networking.IngressLabelKey: name, serving.RouteLabelKey: name},
			},
			Spec: routev1.RouteSpec{Host: "b.example.com"},
		}
	)

	test := struct {
		name  string
		state []routev1.Route
		want  []routev1.Route
	}{
		name: "Clean up old route and new route is generated",

		state: []routev1.Route{{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "to-be-reconciled-0",
				Labels: map[string]string{networking.IngressLabelKey: name, serving.RouteLabelKey: name, serving.RouteNamespaceLabelKey: namespace},
			},
			Spec: routev1.RouteSpec{Host: domainName},
		}, noRemoveMissingLabel, noRemoveOtherLabel},
		want: []routev1.Route{{
			ObjectMeta: metav1.ObjectMeta{
				Name:        routeName0,
				Namespace:   serviceMeshNamespace,
				Labels:      map[string]string{networking.IngressLabelKey: name, serving.RouteLabelKey: name, serving.RouteNamespaceLabelKey: namespace},
				Annotations: map[string]string{resources.TimeoutAnnotation: "5s", networking.IngressClassAnnotationKey: network.IstioIngressClassName},
			},
			Spec: routev1.RouteSpec{
				Host: domainName,
				To: routev1.RouteTargetReference{
					Kind: "Service",
					Name: "istio-ingressgateway",
				},
				Port: &routev1.RoutePort{
					TargetPort: intstr.FromString(resources.KourierHttpPort),
				},
				TLS: &routev1.TLSConfig{
					Termination:                   routev1.TLSTerminationEdge,
					InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyAllow,
				},
			},
		}, noRemoveMissingLabel, noRemoveOtherLabel},
	}

	t.Run(test.name, func(t *testing.T) {
		ingress := defaultIngress.DeepCopy()

		// Register operator types with the runtime scheme.
		s := scheme.Scheme
		s.AddKnownTypes(networkingv1alpha1.SchemeGroupVersion, ingress)
		s.AddKnownTypes(routev1.SchemeGroupVersion, &routev1.Route{})
		s.AddKnownTypes(routev1.SchemeGroupVersion, &routev1.RouteList{})

		// Create a fake client to mock API calls.
		cl := fake.NewFakeClient(ingress, &routev1.RouteList{Items: test.state})

		// Create a Reconcile Ingress object with the scheme and fake client.
		r := &ReconcileIngress{client: cl, scheme: s}
		// Mock request to simulate Reconcile() being called on an event for a
		// watched resource .
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			},
		}
		if _, err := r.Reconcile(req); err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}

		routeList := &routev1.RouteList{}
		err := cl.List(context.TODO(), &client.ListOptions{}, routeList)
		assert.Nil(t, err)

		routes := routeList.Items
		assert.ElementsMatch(t, routes, test.want)

		// Updating ingress with DeletionTimestamp instead of cl.Delete because delete operation doesn't handle finalizers properly.
		ingress.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		if err := cl.Update(context.TODO(), ingress); err != nil {
			t.Fatalf("failed to delete ingress: (%v)", err)
		}

		s.AddKnownTypes(networkingv1alpha1.SchemeGroupVersion, &networkingv1alpha1.IngressList{})
		if _, err := r.Reconcile(req); err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}

		// check openshift routes has been removed.
		routeListdelete := &routev1.RouteList{}
		err = cl.List(context.TODO(), &client.ListOptions{}, routeListdelete)
		assert.Nil(t, err)
		assert.ElementsMatch(t, routeListdelete.Items, []routev1.Route{noRemoveOtherLabel, noRemoveMissingLabel})

		// check finalizers has been removed from ingress.
		ingressListdelete := &networkingv1alpha1.IngressList{}
		err = cl.List(context.TODO(), &client.ListOptions{}, ingressListdelete)
		assert.Nil(t, err)
		assert.Empty(t, len(ingressListdelete.Items[0].Finalizers))
	})
}

// TestIngressController runs Reconcile ReconcileIngress.Reconcile() against a
// fake client that tracks an Ingress object.
func TestIngressController(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	tests := []struct {
		name              string
		annotations       map[string]string
		want              map[string]string
		wantRouteErr      func(err error) bool
		wantNetworkPolicy bool // ServiceMesh required NetworkPolicy but it is no longer necessary. Now we confirm that NetworkPolicy knative-serving-allow-all is always deleted.
		deleted           bool
		extraObjs         []runtime.Object
	}{
		{
			name:              "reconcile route with timeout annotation",
			annotations:       map[string]string{},
			want:              map[string]string{resources.TimeoutAnnotation: "5s", networking.IngressClassAnnotationKey: network.IstioIngressClassName},
			wantRouteErr:      func(err error) bool { return err == nil },
			wantNetworkPolicy: false,
			deleted:           false,
		},
		{
			name:              "reconcile route with taking over annotations",
			annotations:       map[string]string{serving.CreatorAnnotation: "userA", serving.UpdaterAnnotation: "userB"},
			want:              map[string]string{serving.CreatorAnnotation: "userA", serving.UpdaterAnnotation: "userB", resources.TimeoutAnnotation: "5s", networking.IngressClassAnnotationKey: network.IstioIngressClassName},
			wantRouteErr:      func(err error) bool { return err == nil },
			wantNetworkPolicy: false,
			deleted:           false,
		},
		{
			name:              "do not reconcile with disable route annotation",
			annotations:       map[string]string{resources.DisableRouteAnnotation: ""},
			want:              nil,
			wantRouteErr:      errors.IsNotFound,
			wantNetworkPolicy: false,
			deleted:           false,
		},
		{
			name:              "reconcile route with different ingress annotation",
			annotations:       map[string]string{networking.IngressClassAnnotationKey: "kourier"},
			want:              map[string]string{networking.IngressClassAnnotationKey: "kourier", resources.TimeoutAnnotation: "5s"},
			wantRouteErr:      func(err error) bool { return err == nil },
			wantNetworkPolicy: false,
			deleted:           false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ingress := defaultIngress.DeepCopy()

			// Set test annotations
			annotations := ingress.GetAnnotations()
			for k, v := range test.annotations {
				annotations[k] = v
			}
			ingress.SetAnnotations(annotations)

			if test.deleted {
				deletedTime := metav1.Now()
				ingress.SetDeletionTimestamp(&deletedTime)
			}

			// route object
			route := &routev1.Route{}

			initObjs := []runtime.Object{ingress, route}
			initObjs = append(initObjs, test.extraObjs...)

			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(networkingv1alpha1.SchemeGroupVersion, ingress)
			s.AddKnownTypes(routev1.SchemeGroupVersion, route)
			s.AddKnownTypes(routev1.SchemeGroupVersion, &routev1.RouteList{})
			// Create a fake client to mock API calls.
			cl := fake.NewFakeClient(initObjs...)
			// Create a Reconcile Ingress object with the scheme and fake client.
			r := &ReconcileIngress{client: cl, scheme: s}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			}
			if _, err := r.Reconcile(req); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// Check if route has been created.
			routes := &routev1.Route{}
			err := cl.Get(context.TODO(), types.NamespacedName{Name: routeName0, Namespace: serviceMeshNamespace}, routes)

			assert.True(t, test.wantRouteErr(err))
			assert.Equal(t, test.want, routes.ObjectMeta.Annotations)
		})
	}
}
