package e2e

import (
	goctx "context"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestKnativeServing(t *testing.T) {
	knativeServingList := &servingv1alpha1.KnativeServingList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KnativeServing",
			APIVersion: "operator.knative.dev/v1alpha1",
		},
	}

	err := framework.AddToFrameworkScheme(servingv1alpha1.SchemeBuilder.AddToScheme, knativeServingList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	err = e2eutil.WaitForDeployment(t, framework.Global.KubeClient, namespace, "knative-serving-openshift", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("invalid", func(t *testing.T) {
		invalidNamespace(t, ctx)
	})
	t.Run("config", func(t *testing.T) {
		verifyDefaults(t, ctx)
	})
}

func invalidNamespace(t *testing.T, ctx *framework.TestCtx) {
	namespace, _ := ctx.GetNamespace()
	f := framework.Global
	// create KnativeServing custom resource
	knativeServing := &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving",
			Namespace: namespace,
		},
	}
	err := f.Client.Create(goctx.TODO(), knativeServing, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	sErr, ok := err.(*apierrors.StatusError)
	if !(ok && strings.Contains(sErr.Status().Message, "validating.knativeserving.openshift.io")) {
		t.Fatalf("Create should fail in invalid namespace; err => %s", err)
	}
}

func verifyDefaults(t *testing.T, ctx *framework.TestCtx) {
	const namespace = "knative-serving"
	f := framework.Global
	ns := &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	f.Client.Create(goctx.TODO(), ns, &framework.CleanupOptions{TestContext: ctx})
	// create KnativeServing custom resource
	knativeServing := &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving",
			Namespace: namespace,
		},
	}
	err := f.Client.Create(goctx.TODO(), knativeServing, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}
	if len(knativeServing.Spec.Config["domain"]) == 0 {
		t.Fatal("Ingress not set")
	}
	if len(knativeServing.Spec.Config["network"]) == 0 {
		t.Fatal("Egress not set")
	}
}
