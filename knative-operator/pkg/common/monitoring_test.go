package common

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const installedNS = "openshift-serverless"

var (
	operatorNamespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{ Name: installedNS },
	}
	serverlessDeployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "knative-openshift",
			Namespace: installedNS,
		},
	}
)

func init() {
	os.Setenv(installedNamespaceEnvKey, installedNS)
	os.Setenv(operatorDeploymentNameEnvKey, "knative-openshift")
	os.Setenv(testRolePath, "testdata/role_service_monitor.yaml")
}

func TestSetUpMonitoringRequirements(t *testing.T) {
	initObjs := []runtime.Object{&operatorNamespace, &serverlessDeployment}
	cl := fake.NewFakeClient(initObjs...)
	err := SetUpMonitoringRequirements(cl)
	if err != nil {
		t.Errorf("Failed to set up monitoring requirements: %w", err)
	}
	ns := corev1.Namespace{}
	err = cl.Get(context.TODO(), client.ObjectKey{Name: installedNS}, &ns)
	if err != nil {
		t.Errorf("Failed to get modified namespace: %w", err)
	}
	if actual := ns.Labels[monitoringLabel]; actual != "true" {
		t.Errorf("got %q, want %q", actual, "true")
	}
	role := v1.Role{}
	err = cl.Get(context.TODO(), client.ObjectKey{Name: "knative-serving-prometheus-k8s", Namespace: installedNS}, &role)
	if err != nil {
		t.Errorf("Failed to get created role: %w", err)
	}
	if len(role.Rules) == 0 {
		t.Error("Rules should be non emtpy")
	}
	rb := v1.RoleBinding{}
	err = cl.Get(context.TODO(), client.ObjectKey{Name: "knative-serving-prometheus-k8s", Namespace: installedNS}, &rb)
	if err != nil {
		t.Errorf("Failed to get created rolebinding: %w", err)
	}
	if len(rb.Subjects) == 0 {
		t.Error("Subjects should be non emtpy")
	}
	sub := rb.Subjects[0]
	if sub.Kind != "ServiceAccount" {
		t.Errorf("got %q, want %q", sub.Kind, "ServiceAccount")
	}
	if sub.Name != "prometheus-k8s" {
		t.Errorf("got %q, want %q", sub.Kind, "prometheus-k8s")
	}
	if sub.Namespace != "openshift-monitoring" {
		t.Errorf("got %q, want %q", sub.Kind, "openshift-monitoring")
	}
}

