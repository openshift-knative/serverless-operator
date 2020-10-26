package sources

import (
	"context"
	"os"
	"testing"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	apiserverRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "api1"},
	}

	pingsourceRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: "knative-eventing", Name: "ping1"},
	}

	apiserversourceDeployment = v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api1",
			Namespace: "default",
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.SourceLabel:     "apiserver-source-controller",
					common.SourceNameLabel: "api1",
				},
			},
		},
	}
	pingsourceDeployment = v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ping1",
			Namespace: "knative-eventing",
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.SourceLabel:     "ping-source-controller",
					common.SourceRoleLabel: "adapter",
				},
			},
		},
	}
	defaultNamespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
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
	os.Setenv(common.TestRolePath, "../testdata/role_service_monitor.yaml")
	os.Setenv(common.TestSourceServiceMonitorPath, "../testdata/source-service-monitor.yaml")
	os.Setenv(common.TestSourceServicePath, "../testdata/source-service.yaml")

	apis.AddToScheme(scheme.Scheme)
}

// TestSourceReconcile runs Reconcile to verify if monitoring resources are created/deleted for sources.
func TestSourceReconcile(t *testing.T) {
	initObjs := []runtime.Object{&apiserversourceDeployment, &pingsourceDeployment, &defaultNamespace, &eventingNamespace}
	cl := fake.NewFakeClient(initObjs...)

	r := &ReconcileSourceDeployment{client: cl, scheme: scheme.Scheme}
	// Reconcile for an api server source
	if _, err := r.Reconcile(apiserverRequest); err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	smAPIService := &corev1.Service{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: apiserverRequest.Name, Namespace: apiserverRequest.Namespace}, smAPIService); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if smAPIService.Spec.Selector[common.SourceLabel] != "apiserver-source-controller" {
		t.Fatalf("got %q, want %q", smAPIService.Spec.Selector[common.SourceLabel], "apiserver-source-controller")
	}
	if smAPIService.Spec.Selector[common.SourceNameLabel] != "api1" {
		t.Fatalf("got %q, want %q", smAPIService.Spec.Selector[common.SourceNameLabel], "api1")
	}
	smAPI := &monitoringv1.ServiceMonitor{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: apiserverRequest.Name, Namespace: apiserverRequest.Namespace}, smAPI); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if smAPI.Spec.Selector.MatchLabels["name"] != "api1" {
		t.Fatalf("got %q, want %q", smAPI.Spec.Selector.MatchLabels["name"], "api1")
	}

	// Reconcile for a ping source
	if _, err := r.Reconcile(pingsourceRequest); err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	smPingService := &corev1.Service{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: pingsourceRequest.Name, Namespace: pingsourceRequest.Namespace}, smPingService); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if smPingService.Spec.Selector[common.SourceLabel] != "ping-source-controller" {
		t.Fatalf("got %q, want %q", smPingService.Spec.Selector[common.SourceLabel], "ping-source-controller")
	}
	if smPingService.Spec.Selector[common.SourceRoleLabel] != "adapter" {
		t.Fatalf("got %q, want %q", smPingService.Spec.Selector[common.SourceRoleLabel], "adapter")
	}
	smPing := &monitoringv1.ServiceMonitor{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: pingsourceRequest.Name, Namespace: pingsourceRequest.Namespace}, smPing); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if smPing.Spec.Selector.MatchLabels["name"] != "ping1" {
		t.Fatalf("got %q, want %q", smPing.Spec.Selector.MatchLabels["name"], "ping1")
	}
}
