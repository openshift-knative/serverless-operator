package knativeeventing

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/dashboard"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	defaultKnativeEventing = v1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-eventing",
			Namespace: "knative-eventing",
		},
	}
	defaultRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: "knative-eventing", Name: "knative-eventing"},
	}
	dashboardNamespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: dashboard.ConfigManagedNamespace,
		},
	}
	eventingNamespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "knative-eventing",
		},
	}
)

func init() {
	os.Setenv("OPERATOR_NAME", "TEST_OPERATOR")
	os.Setenv(dashboard.EventingSourceDashboardPathEnvVar, "../dashboard/testdata/grafana-dash-knative-eventing-source.yaml")
	os.Setenv(dashboard.EventingBrokerDashboardPathEnvVar, "../dashboard/testdata/grafana-dash-knative-eventing-broker.yaml")
	os.Setenv(common.TestRolePath, "../dashboard/testdata/role_service_monitor.yaml")
	os.Setenv(common.TestEventingBrokerServiceMonitorPath, "../dashboard/testdata/broker-service-monitors.yaml")
	os.Setenv(common.TestMonitor, "true")
}

// TestEventingReconcile runs Reconcile to verify if eventing resources are created/deleted.
func TestEventingReconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	tests := []struct {
		name           string
		ownerName      string
		ownerNamespace string
		deleted        bool
	}{
		{
			name:           "reconcile request with same KnativeServing owner",
			ownerName:      "knative-eventing",
			ownerNamespace: "knative-eventing",
			deleted:        true,
		},
		{
			name:           "reconcile request with different KnativeServing owner name",
			ownerName:      "FOO",
			ownerNamespace: "knative-eventing",
			deleted:        false,
		},
		{
			name:           "reconcile request with different KnativeServing owner namespace",
			ownerName:      "knative-eventing",
			ownerNamespace: "FOO",
			deleted:        false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ke := &defaultKnativeEventing
			ns := &dashboardNamespace
			monitor := &monitoringv1.ServiceMonitor{}
			initObjs := []runtime.Object{ke, ns, &eventingNamespace}

			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, ke)
			scheme.Scheme.AddKnownTypes(monitoringv1.SchemeGroupVersion, monitor)

			cl := fake.NewFakeClient(initObjs...)
			r := &ReconcileKnativeEventing{client: cl, scheme: s}

			// Reconcile to initialize
			if _, err := r.Reconcile(defaultRequest); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}
			// Check if Eventing dashboard configmaps are available
			brokerCM := &corev1.ConfigMap{}
			err := cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-broker", Namespace: ns.Name}, brokerCM)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}
			sourceCM := &corev1.ConfigMap{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-source", Namespace: ns.Name}, sourceCM)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}
			// Check if the eventing service monitors are installed
			smFilter := &monitoringv1.ServiceMonitor{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "knative-eventing-metrics-broker-filter", Namespace: ns.Namespace}, smFilter)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}
			smIngress := &monitoringv1.ServiceMonitor{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "knative-eventing-metrics-broker-ingress", Namespace: ns.Namespace}, smIngress)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}
			// Delete Dashboard configmaps.
			err = cl.Delete(context.TODO(), brokerCM)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}
			err = cl.Delete(context.TODO(), sourceCM)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}
			// Delete service monitors
			err = cl.Delete(context.TODO(), smFilter)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}
			err = cl.Delete(context.TODO(), smIngress)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}
			// Reconcile again with test requests.
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: test.ownerNamespace, Name: test.ownerName},
			}
			if _, err := r.Reconcile(req); err != nil {
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
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-broker", Namespace: ns.Name}, brokerCM)
			checkError(t, err)
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative-eventing-source", Namespace: ns.Name}, sourceCM)
			checkError(t, err)

			// Check again if the eventing service monitors are available
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "knative-eventing-metrics-broker-filter", Namespace: ns.Namespace}, smFilter)
			checkError(t, err)
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "knative-eventing-metrics-broker-ingress", Namespace: ns.Namespace}, smIngress)
			checkError(t, err)
		})
	}
}
