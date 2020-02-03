package knativeservingobsolete

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"

	oldapi "github.com/knative/pkg/apis"
	obsolete "github.com/openshift-knative/serverless-operator/serving/operator/pkg/apis/serving/v1alpha1"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

var (
	mockEnv = []runtime.Object{
		&configv1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: configv1.NetworkSpec{
				ServiceNetwork: []string{"foo"},
			},
		},
		&configv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: configv1.IngressSpec{
				Domain: "domain.example",
			},
		},
	}
	key = types.NamespacedName{
		Namespace: "knative-serving",
		Name:      "knative-serving",
	}
)

func TestKnativeServingMigrationMirrorsStatusUp(t *testing.T) {
	ctx := context.TODO()
	s := scheme.Scheme
	apis.AddToScheme(s)

	old := &obsolete.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Namespace,
			Namespace: key.Name,
		},
	}
	objs := append(mockEnv, old)
	cl := fake.NewFakeClient(objs...)

	r := &ReconcileKnativeServingObsolete{client: cl, scheme: s}

	if _, err := r.Reconcile(reconcile.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// The new resource should have been created.
	created := &servingv1alpha1.KnativeServing{}
	if err := cl.Get(ctx, key, created); err != nil {
		t.Fatalf("Failed to get new object: %v", err)
	}

	// Update the status of the new resource...
	created.Status.Version = "v0.11.1"
	created.Status.InitializeConditions()
	created.Status.MarkDependencyMissing("failed")
	if err := cl.Status().Update(ctx, created); err != nil {
		t.Fatalf("Failed to update status initially: %v", err)
	}
	// ...which should be reflected on the old resource after a reconcile.
	if _, err := r.Reconcile(reconcile.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if err := cl.Get(ctx, key, old); err != nil {
		t.Fatalf("Failed to get old object: %v", err)
	}

	want := obsolete.KnativeServingStatus{
		Version: "v0.11.1",
	}
	want.InitializeConditions()
	want.MarkDependencyMissing("failed")
	if diff := cmp.Diff(want, old.Status, cmpopts.IgnoreFields(oldapi.Condition{}, "LastTransitionTime")); diff != "" {
		t.Fatalf("Status was not equal: %s\n", diff)
	}

	// Actually make the new resource ready...
	created.Status.MarkDependenciesInstalled()
	created.Status.MarkDeploymentsAvailable()
	created.Status.MarkInstallSucceeded()
	if err := cl.Status().Update(ctx, created); err != nil {
		t.Fatalf("Failed to update status initially: %v", err)
	}
	// ...which should be reflected on the old resource after a reconcile.
	if _, err := r.Reconcile(reconcile.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if err := cl.Get(ctx, key, old); err != nil {
		t.Fatalf("Failed to get old object: %v", err)
	}
	if !old.Status.IsReady() {
		t.Fatalf("IsReady() = false, want true")
	}
}

func TestKnativeServingMigrationMirrorsConfigDown(t *testing.T) {
	ctx := context.TODO()
	s := scheme.Scheme
	apis.AddToScheme(s)

	old := &obsolete.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Namespace,
			Namespace: key.Name,
		},
		Spec: obsolete.KnativeServingSpec{
			Config: map[string]map[string]string{
				"foo": map[string]string{
					"bar": "baz",
				}},
		},
	}
	objs := append(mockEnv, old)
	cl := fake.NewFakeClient(objs...)

	r := &ReconcileKnativeServingObsolete{client: cl, scheme: s}

	if _, err := r.Reconcile(reconcile.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// The new resource should have been created.
	created := &servingv1alpha1.KnativeServing{}
	if err := cl.Get(ctx, key, created); err != nil {
		t.Fatalf("Failed to get new object: %v", err)
	}
	if created.Spec.Config["foo"]["bar"] != "baz" {
		t.Fatalf("Spec.Config was not as expected: %v", created.Spec.Config)
	}

	// Update status of the new resource to verify it won't be overridden.
	created.Status.Version = "v0.11.1"
	if err := cl.Status().Update(ctx, created); err != nil {
		t.Fatalf("Failed to update status initially: %v", err)
	}
	if _, err := r.Reconcile(reconcile.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Update the config of the old resource and verify it's updated downstream.
	old.Spec.Config["foo"]["bar"] = "baz3"
	if err := cl.Update(ctx, old); err != nil {
		t.Fatalf("Failed to update status initially: %v", err)
	}

	if _, err := r.Reconcile(reconcile.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// The new resource should be updated.
	if err := cl.Get(ctx, key, created); err != nil {
		t.Fatalf("Failed to get new object: %v", err)
	}
	if created.Spec.Config["foo"]["bar"] != "baz3" {
		t.Fatalf("Spec.Config was not as expected: %v", created.Spec.Config)
	}
	// Verify the status didn't get thrown away.
	if created.Status.Version != "v0.11.1" {
		t.Fatalf("Spec.Status got thrown away unexpectedly")
	}

	// Add a new field upstream.
	old.Spec.Config["foo2"] = map[string]string{
		"bar2": "baz2",
	}
	if err := cl.Update(ctx, old); err != nil {
		t.Fatalf("Failed to update status initially: %v", err)
	}

	if _, err := r.Reconcile(reconcile.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// The new resource should be updated.
	if err := cl.Get(ctx, key, created); err != nil {
		t.Fatalf("Failed to get new object: %v", err)
	}
	if created.Spec.Config["foo"]["bar"] != "baz3" || created.Spec.Config["foo2"]["bar2"] != "baz2" {
		t.Fatalf("Spec.Config was not as expected: %v", created.Spec.Config)
	}
	// Verify the status didn't get thrown away.
	if created.Status.Version != "v0.11.1" {
		t.Fatalf("Spec.Status got thrown away unexpectedly")
	}
}
