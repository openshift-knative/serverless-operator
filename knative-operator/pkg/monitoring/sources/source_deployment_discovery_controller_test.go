package sources

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	okomon "github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	apiserversourceDeployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api1",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					SourceLabel:     "apiserver-source-controller",
					SourceNameLabel: "api1",
				},
			},
		},
	}
	pingsourceDeployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ping1",
			Namespace: "knative-eventing",
		},
		Spec: appsv1.DeploymentSpec{
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
	eventingInstance := &operatorv1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-eventing",
			Namespace: "knative-eventing",
		},
	}
	keUpdate(eventingInstance, func(ke *operatorv1beta1.KnativeEventing) {
		common.Configure(&ke.Spec.CommonSpec, okomon.ObservabilityCMName, okomon.ObservabilityBackendKey, "prometheus")
	})
	cl := fake.NewClientBuilder().
		WithObjects(&apiserversourceDeployment, &pingsourceDeployment, &defaultNamespace, &eventingNamespace, eventingInstance).
		Build()
	r := &ReconcileSourceDeployment{client: cl, scheme: scheme.Scheme}
	_ = os.Setenv(generateSourceServiceMonitorsEnvVar, "true")
	defer os.Unsetenv(generateSourceServiceMonitorsEnvVar)
	_ = os.Setenv(useClusterMonitoringEnvVar, "true")
	defer os.Unsetenv(useClusterMonitoringEnvVar)
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
	checkPrometheusResources(cl, true, t)
	checkSourceServiceMonitors(cl, true, apiserverRequest.Name, apiserverRequest.Namespace, t)
}

func TestSourceMonitoringReconcile(t *testing.T) {
	eventingInstance := &operatorv1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-eventing",
			Namespace: "knative-eventing",
		},
	}
	cl := fake.NewClientBuilder().
		WithObjects(&apiserversourceDeployment, &pingsourceDeployment, &defaultNamespace, &eventingNamespace, eventingInstance).
		Build()

	// No cluster monitoring only generate service monitors.
	r := &ReconcileSourceDeployment{client: cl, scheme: scheme.Scheme}
	_ = os.Setenv(generateSourceServiceMonitorsEnvVar, "true")
	defer os.Unsetenv(generateSourceServiceMonitorsEnvVar)
	_ = os.Setenv(useClusterMonitoringEnvVar, "false")
	defer os.Unsetenv(useClusterMonitoringEnvVar)
	// Reconcile for an api server source
	if _, err := r.Reconcile(context.Background(), apiserverRequest); err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	ns := &corev1.Namespace{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: apiserverRequest.Namespace}, ns); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if ns.Labels[okomon.EnableMonitoringLabel] != "false" {
		t.Fatalf("got %q, want %q", ns.Labels[okomon.EnableMonitoringLabel], "false")
	}
	checkPrometheusResources(cl, false, t)
	newEventingInstance := eventingInstance.DeepCopy()
	newEventingInstance = keUpdate(newEventingInstance, func(ke *operatorv1beta1.KnativeEventing) {
		common.Configure(&ke.Spec.CommonSpec, okomon.ObservabilityCMName, okomon.ObservabilityBackendKey, "none")
	})
	if err := cl.Update(context.TODO(), newEventingInstance); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if _, err := r.Reconcile(context.Background(), apiserverRequest); err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	ns = &corev1.Namespace{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: apiserverRequest.Namespace}, ns); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if ns.Labels[okomon.EnableMonitoringLabel] != "false" {
		t.Fatalf("got %q, want %q", ns.Labels[okomon.EnableMonitoringLabel], "false")
	}
	checkPrometheusResources(cl, false, t)
	r = &ReconcileSourceDeployment{client: cl, scheme: scheme.Scheme}
	_ = os.Setenv(generateSourceServiceMonitorsEnvVar, "false")
	_ = os.Setenv(useClusterMonitoringEnvVar, "false")
	// Reconcile for an api server source
	if _, err := r.Reconcile(context.Background(), apiserverRequest); err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	ns = &corev1.Namespace{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: apiserverRequest.Namespace}, ns); err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if ns.Labels[okomon.EnableMonitoringLabel] != "false" {
		t.Fatalf("got %q, want %q", ns.Labels[okomon.EnableMonitoringLabel], "false")
	}
	checkPrometheusResources(cl, false, t)
	checkSourceServiceMonitors(cl, false, apiserverRequest.Name, apiserverRequest.Namespace, t)
}

func checkPrometheusResources(cl client.Client, shouldExist bool, t *testing.T) {
	role := &rbacv1.Role{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: "knative-prometheus-k8s", Namespace: apiserverRequest.Namespace}, role); checkError(err, shouldExist, t) {
		t.Fatalf("get: (%v)", err)
	}
	roleBinding := &rbacv1.RoleBinding{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: "knative-prometheus-k8s", Namespace: apiserverRequest.Namespace}, roleBinding); checkError(err, shouldExist, t) {
		t.Fatalf("get: (%v)", err)
	}
}

func checkSourceServiceMonitors(cl client.Client, shouldExist bool, name string, ns string, t *testing.T) {
	sm := &monitoringv1.ServiceMonitor{}
	if err := cl.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, sm); checkError(err, shouldExist, t) {
		t.Fatalf("get: (%v)", err)
	}
}

func checkError(err error, shouldExist bool, t *testing.T) bool {
	if shouldExist {
		if err != nil {
			return true
		}
	} else {
		if err != nil {
			return !apierrors.IsNotFound(err)
		}
		t.Fatal("Resource should not exist")
	}
	return false
}

func keUpdate(instance *operatorv1beta1.KnativeEventing, mods ...func(*operatorv1beta1.KnativeEventing)) *operatorv1beta1.KnativeEventing {
	for _, mod := range mods {
		mod(instance)
	}
	return instance
}
