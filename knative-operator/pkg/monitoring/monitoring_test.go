package monitoring

import (
	"context"
	"os"
	"testing"

	okomon "github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const installedNS = "openshift-serverless"

var (
	operatorNamespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: installedNS},
	}
	serverlessDeployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-openshift",
			Namespace: installedNS,
		},
	}
)

func init() {
	_ = monitoringv1.AddToScheme(scheme.Scheme)
	os.Setenv(operatorDeploymentNameEnvKey, "knative-openshift")
}

func TestSetupMonitoringRequirements(t *testing.T) {
	cl := fake.NewClientBuilder().WithObjects(&operatorNamespace, &serverlessDeployment).Build()
	err := SetupClusterMonitoringRequirements(cl, &serverlessDeployment, serverlessDeployment.GetNamespace(), nil)
	if err != nil {
		t.Errorf("Failed to set up monitoring requirements: %w", err)
	}
	ns := corev1.Namespace{}
	err = cl.Get(context.TODO(), client.ObjectKey{Name: installedNS}, &ns)
	if err != nil {
		t.Errorf("Failed to get modified namespace: %w", err)
	}
	if actual := ns.Labels[okomon.EnableMonitoringLabel]; actual != "true" {
		t.Errorf("got %q, want %q", actual, "true")
	}
	role := v1.Role{}
	err = cl.Get(context.TODO(), client.ObjectKey{Name: rbacName, Namespace: installedNS}, &role)
	if err != nil {
		t.Errorf("Failed to get created role: %w", err)
	}
	if len(role.Rules) == 0 {
		t.Error("Rules should be non empty")
	}
	rb := v1.RoleBinding{}
	err = cl.Get(context.TODO(), client.ObjectKey{Name: rbacName, Namespace: installedNS}, &rb)
	if err != nil {
		t.Errorf("Failed to get created rolebinding: %w", err)
	}
	if len(rb.Subjects) == 0 {
		t.Error("Subjects should be non empty")
	}
	sub := rb.Subjects[0]
	if sub.Kind != "ServiceAccount" {
		t.Errorf("got %q, want %q", sub.Kind, "ServiceAccount")
	}
	if sub.Name != "prometheus-k8s" {
		t.Errorf("got %q, want %q", sub.Name, "prometheus-k8s")
	}
	if sub.Namespace != "openshift-monitoring" {
		t.Errorf("got %q, want %q", sub.Namespace, "openshift-monitoring")
	}
}

func TestRemoveOldServiceMonitorResources(t *testing.T) {
	oldSM := monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace.Name,
			Name:      "knative-openshift-metrics",
		},
	}
	oldSMService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace.Name,
			Name:      "knative-openshift-metrics",
		},
	}
	newSM := monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace.Name,
			Name:      "knative-openshift-metrics-3",
		},
	}
	newSMService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace.Name,
			Name:      "knative-openshift-metrics-3",
		},
	}
	randomSM := monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace.Name,
			Name:      "random",
		},
	}
	randomService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace.Name,
			Name:      "random",
		},
	}
	initObjs := []client.Object{&operatorNamespace, &oldSM, &oldSMService, &newSM, &newSMService, &randomSM, &randomService}
	cl := fake.NewClientBuilder().WithObjects(initObjs...).Build()
	if err := RemoveOldServiceMonitorResourcesIfExist(operatorNamespace.Name, cl); err != nil {
		t.Errorf("Failed to remove old service monitor resources: %w", err)
	}
	smList := monitoringv1.ServiceMonitorList{}
	if err := cl.List(context.TODO(), &smList, client.InNamespace(operatorNamespace.Name)); err != nil {
		t.Errorf("Failed to list available service monitors: %w", err)
	}
	if len(smList.Items) != 2 {
		t.Errorf("got %d, want %d", len(smList.Items), 2)
	}
	for _, sm := range smList.Items {
		if sm.Name != "knative-openshift-metrics-3" && sm.Name != "random" {
			t.Errorf("got %q, want %q", sm.Name, "knative-openshift-metrics-3 or random")
		}
	}
	smServiceList := corev1.ServiceList{}
	if err := cl.List(context.TODO(), &smServiceList, client.InNamespace(operatorNamespace.Name)); err != nil {
		t.Errorf("Failed to list available services: %w", err)
	}
	if len(smServiceList.Items) != 2 {
		t.Errorf("got %d, want %d", len(smServiceList.Items), 2)
	}
	for _, sv := range smServiceList.Items {
		if sv.Name != "knative-openshift-metrics-3" && sv.Name != "random" {
			t.Errorf("got %q, want %q", sv.Name, "knative-openshift-metrics-3 or random")
		}
	}
}
