package sources

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					SourceLabel:     "apiserver-source-controller",
					SourceNameLabel: "api1",
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
					SourceLabel:     "ping-source-controller",
					SourceRoleLabel: "adapter",
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
	apis.AddToScheme(scheme.Scheme)
}

// TestSourceReconcile runs Reconcile to verify if monitoring resources are created/deleted for sources.
func TestSourceReconcile(t *testing.T) {
	cl := fake.NewClientBuilder().
		WithObjects(&apiserversourceDeployment, &pingsourceDeployment, &defaultNamespace, &eventingNamespace).
		Build()

	r := &ReconcileSourceDeployment{client: cl, scheme: scheme.Scheme}
	// Reconcile for an api server source
	if _, err := r.Reconcile(context.Background(), apiserverRequest); err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	smAPIService := &corev1.Service{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: apiserverRequest.Name, Namespace: apiserverRequest.Namespace}, smAPIService); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if smAPIService.Spec.Selector[SourceLabel] != "apiserver-source-controller" {
		t.Fatalf("got %q, want %q", smAPIService.Spec.Selector[SourceLabel], "apiserver-source-controller")
	}
	if smAPIService.Spec.Selector[SourceNameLabel] != "api1" {
		t.Fatalf("got %q, want %q", smAPIService.Spec.Selector[SourceNameLabel], "api1")
	}
	smAPI := &monitoringv1.ServiceMonitor{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: apiserverRequest.Name, Namespace: apiserverRequest.Namespace}, smAPI); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if smAPI.Spec.Selector.MatchLabels["name"] != "api1" {
		t.Fatalf("got %q, want %q", smAPI.Spec.Selector.MatchLabels["name"], "api1")
	}

	// Reconcile for a ping source
	if _, err := r.Reconcile(context.Background(), pingsourceRequest); err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	smPingService := &corev1.Service{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: pingsourceRequest.Name, Namespace: pingsourceRequest.Namespace}, smPingService); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if smPingService.Spec.Selector[SourceLabel] != "ping-source-controller" {
		t.Fatalf("got %q, want %q", smPingService.Spec.Selector[SourceLabel], "ping-source-controller")
	}
	if smPingService.Spec.Selector[SourceRoleLabel] != "adapter" {
		t.Fatalf("got %q, want %q", smPingService.Spec.Selector[SourceRoleLabel], "adapter")
	}
	smPing := &monitoringv1.ServiceMonitor{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: pingsourceRequest.Name, Namespace: pingsourceRequest.Namespace}, smPing); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if smPing.Spec.Selector.MatchLabels["name"] != "ping1" {
		t.Fatalf("got %q, want %q", smPing.Spec.Selector.MatchLabels["name"], "ping1")
	}
}
