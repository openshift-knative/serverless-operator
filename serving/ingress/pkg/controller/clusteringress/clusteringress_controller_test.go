package clusteringress

import (
	"context"
	"testing"
	"time"

	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	"github.com/stretchr/testify/assert"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/controller/common"
	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/controller/resources"

	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/serving/pkg/apis/networking"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/network"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// TestClusterIngressController runs Reconcile ReconcileClusterIngress.Reconcile() against a
// fake client that tracks a ClusteIngress object.
func TestClusterIngressController(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	var (
		name                 = "clusteringress-operator"
		serviceMeshNamespace = "knative-serving-ingress"
		smmrName             = "default"
		namespace            = "clusteringress-namespace"
		uid                  = "8a7e9a9d-fbc6-11e9-a88e-0261aff8d6d8"
		domainName           = name + "." + namespace + ".default.domainName"
		routeName0           = "route-" + uid + "-303036363463"
	)

	// A ServiceMeshMemberRole resource with metadata
	smmr := &maistrav1.ServiceMeshMemberRoll{
		ObjectMeta: metav1.ObjectMeta{
			Name:      smmrName,
			Namespace: serviceMeshNamespace,
		},
	}
	// A ClusterIngress resource with metadata and spec.
	clusteringress := &networkingv1alpha1.ClusterIngress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			UID:         types.UID(uid),
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

	// route object
	route := &routev1.Route{}

	// Objects to track in the fake client.
	objs := []runtime.Object{
		smmr,
		clusteringress,
		route,
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(maistrav1.SchemeGroupVersion, smmr)
	s.AddKnownTypes(networkingv1alpha1.SchemeGroupVersion, clusteringress)
	s.AddKnownTypes(routev1.SchemeGroupVersion, route)
	s.AddKnownTypes(routev1.SchemeGroupVersion, &routev1.RouteList{})
	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	// Create a Reconcile ClusterIngress object with the scheme and fake client.
	r := &ReconcileClusterIngress{base: &common.BaseIngressReconciler{Client: cl}, client: cl, scheme: s}

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

	// Check if namespace has been added to smmr
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: smmrName, Namespace: serviceMeshNamespace}, smmr); err != nil {
		t.Fatalf("failed to get ServiceMeshMemberRole: (%v)", err)
	}
	assert.Equal(t, []string{namespace}, smmr.Spec.Members)
	// Check if route has been created
	routes := &routev1.Route{}

	if err := cl.Get(context.TODO(), types.NamespacedName{Name: routeName0, Namespace: serviceMeshNamespace}, routes); err != nil {
		t.Fatalf("get route: (%v)", err)
	}

	assert.Equal(t, "5s", routes.ObjectMeta.Annotations[resources.TimeoutAnnotation])
	assert.NotEqual(t, 10*time.Minute, routes.ObjectMeta.Annotations[resources.TimeoutAnnotation])

}
