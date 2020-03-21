package knativeserving

import (
	"context"
	"os"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/apis/istio/v1alpha3"
	"knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	defaultKnativeServing = v1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving",
			Namespace: "knative-serving",
		},
	}
	defaultIngress = configv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.IngressSpec{
			Domain: "example.com",
		},
	}
)

// TestKourierReconcile runs Reconcile to verify if expected Kourier resources are reconciled.
func TestKourierReconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	tests := []struct {
		name           string
		ownerName      string
		ownerNamespace string
		reconciled     bool
	}{
		{
			name:           "reconcile request with same KnativeServing owner",
			ownerName:      "knative-serving",
			ownerNamespace: "knative-serving",
			reconciled:     true,
		},
		{
			name:           "reconcile request with different KnativeServing owner name",
			ownerName:      "FOO",
			ownerNamespace: "knative-serving",
			reconciled:     false,
		},
		{
			name:           "reconcile request with different KnativeServing owner namespace",
			ownerName:      "knative-serving",
			ownerNamespace: "FOO",
			reconciled:     false,
		},
	}

	os.Setenv("OPERATOR_NAME", "TEST_OPERATOR")
	os.Setenv("KOURIER_MANIFEST_PATH", "kourier/testdata/kourier-latest.yaml")
	os.Setenv("CONSOLE_DOWNLOAD_MANIFEST_PATH", "consoleclidownload/testdata/console_cli_download_kn.yaml")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ks := &defaultKnativeServing
			ingress := &defaultIngress

			initObjs := []runtime.Object{ks, ingress}

			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, ks)
			s.AddKnownTypes(configv1.SchemeGroupVersion, ingress)
			s.AddKnownTypes(v1alpha3.SchemeGroupVersion, &v1alpha3.VirtualServiceList{})

			cl := fake.NewFakeClient(initObjs...)
			r := &ReconcileKnativeServing{client: cl, scheme: s}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: "knative-serving", Name: "knative-serving"},
			}
			// Reconcile to intialize
			if _, err := r.Reconcile(req); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// Check if Kourier is deployed.
			deploy := &appsv1.Deployment{}
			err := cl.Get(context.TODO(), types.NamespacedName{Name: "3scale-kourier-gateway", Namespace: "knative-serving-ingress"}, deploy)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}

			// Delete Kourier deployment.
			err = cl.Delete(context.TODO(), deploy)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}

			// Reconcile again with test requests.
			req = reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: test.ownerNamespace, Name: test.ownerName},
			}
			if _, err := r.Reconcile(req); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// Check again if Kourier deployment is created after reconcile.
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "3scale-kourier-gateway", Namespace: "knative-serving-ingress"}, deploy)
			if test.reconciled {
				if err != nil {
					t.Fatalf("get: (%v)", err)
				}
			}
			if !test.reconciled {
				if !errors.IsNotFound(err) {
					t.Fatalf("get: (%v)", err)
				}
			}
		})
	}
}
