package knativeeventing

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/dashboards"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	ke = &v1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-eventing",
			Namespace: "knative-eventing",
		},
	}
	req = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: ke.Namespace, Name: ke.Name},
	}
	dashboardNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: dashboards.ConfigManagedNamespace,
		},
	}
	eventingNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "knative-eventing",
		},
	}
)

func init() {
	os.Setenv("OPERATOR_NAME", "TEST_OPERATOR")
	os.Setenv(dashboards.DashboardsManifestPathEnvVar, "../../../deploy/resources/dashboards")

	apis.AddToScheme(scheme.Scheme)
}

// TestEventingReconcile runs Reconcile to verify if eventing resources are created/deleted.
func TestEventingReconcile(t *testing.T) {
	tests := []struct {
		name           string
		ownerName      string
		ownerNamespace string
		deleted        bool
	}{{
		name:           "reconcile request with same KnativeEventing owner",
		ownerName:      "knative-eventing",
		ownerNamespace: "knative-eventing",
		deleted:        true,
	}, {
		name:           "reconcile request with different KnativeEventing owner name",
		ownerName:      "FOO",
		ownerNamespace: "knative-eventing",
		deleted:        false,
	}, {
		name:           "reconcile request with different KnativeEventing owner namespace",
		ownerName:      "knative-eventing",
		ownerNamespace: "FOO",
		deleted:        false,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cl := fake.NewClientBuilder().WithObjects(ke, dashboardNamespace, eventingNamespace).Build()
			r := &ReconcileKnativeEventing{client: cl, scheme: scheme.Scheme}
			// Reconcile to initialize
			if _, err := r.Reconcile(context.Background(), req); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}
			// Check if Eventing dashboard configmaps are available
			resourcesCM := &corev1.ConfigMap{}
			err := cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-resources", Namespace: dashboardNamespace.Name}, resourcesCM)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}
			brokerCM := &corev1.ConfigMap{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-broker", Namespace: dashboardNamespace.Name}, brokerCM)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}
			sourceCM := &corev1.ConfigMap{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-source", Namespace: dashboardNamespace.Name}, sourceCM)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}
			channelCM := &corev1.ConfigMap{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-channel", Namespace: dashboardNamespace.Name}, channelCM)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}
			// Delete Dashboard configmaps.
			err = cl.Delete(context.TODO(), resourcesCM)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}
			err = cl.Delete(context.TODO(), brokerCM)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}
			err = cl.Delete(context.TODO(), sourceCM)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}
			err = cl.Delete(context.TODO(), channelCM)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}
			// Reconcile again with test requests.
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: test.ownerNamespace, Name: test.ownerName},
			}
			if _, err := r.Reconcile(context.Background(), req); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}
			var checkError = func(t *testing.T, err error) {
				if test.deleted {
					if err != nil {
						t.Fatalf("get: (%v)", err)
					}
				}
				if !test.deleted {
					if !errors.IsNotFound(err) {
						t.Fatalf("get: (%v)", err)
					}
				}
			}
			// Check again if Eventing dashboard configmaps are available
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-resources", Namespace: dashboardNamespace.Name}, resourcesCM)
			checkError(t, err)
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-broker", Namespace: dashboardNamespace.Name}, brokerCM)
			checkError(t, err)
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-source", Namespace: dashboardNamespace.Name}, sourceCM)
			checkError(t, err)
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-channel", Namespace: dashboardNamespace.Name}, sourceCM)
			checkError(t, err)
		})
	}
}
