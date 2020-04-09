package knativeserving

import (
	"context"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

	defaultVirtualService = v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsName",
			Namespace: "vsNamespace",
		},
	}

	defaultRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: "knative-serving", Name: "knative-serving"},
	}
)

func init() {
	os.Setenv("OPERATOR_NAME", "TEST_OPERATOR")
	os.Setenv("KOURIER_MANIFEST_PATH", "kourier/testdata/kourier-latest.yaml")
	os.Setenv("CONSOLE_DOWNLOAD_MANIFEST_PATH", "consoleclidownload/testdata/console_cli_download_kn.yaml")
}

// TestKourierReconcile runs Reconcile to verify if expected Kourier resources are deleted.
func TestKourierReconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	tests := []struct {
		name           string
		ownerName      string
		ownerNamespace string
		deleted        bool
	}{
		{
			name:           "reconcile request with same KnativeServing owner",
			ownerName:      "knative-serving",
			ownerNamespace: "knative-serving",
			deleted:        true,
		},
		{
			name:           "reconcile request with different KnativeServing owner name",
			ownerName:      "FOO",
			ownerNamespace: "knative-serving",
			deleted:        false,
		},
		{
			name:           "reconcile request with different KnativeServing owner namespace",
			ownerName:      "knative-serving",
			ownerNamespace: "FOO",
			deleted:        false,
		},
	}

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

			// Reconcile to intialize
			if _, err := r.Reconcile(defaultRequest); err != nil {
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
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: test.ownerNamespace, Name: test.ownerName},
			}
			if _, err := r.Reconcile(req); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// Check again if Kourier deployment is created after reconcile.
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "3scale-kourier-gateway", Namespace: "knative-serving-ingress"}, deploy)
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
		})
	}
}

// TestKourierReconcile runs Reconcile to verify if orphaned virtualservice is deleted or not
func TestDeleteVirtualServiceReconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	tests := []struct {
		name        string
		labels      map[string]string
		annotations map[string]string
		deleted     bool
	}{
		{
			name:        "delete virtualservice with expected label and annotation",
			labels:      map[string]string{routeLabelKey: "something", "a": "b"},
			annotations: map[string]string{ingressClassKey: istioIngressClass, "c": "d"},
			deleted:     true,
		},
		{
			name:        "do not delete virtualservice with expected label but without annotation",
			labels:      map[string]string{routeLabelKey: "something", "a": "b"},
			annotations: map[string]string{"c": "d"},
			deleted:     false,
		},
		{
			name:        "do not delete virtualservice with expected annotation but without label",
			labels:      map[string]string{"a": "b"},
			annotations: map[string]string{ingressClassKey: istioIngressClass, "c": "d"},
			deleted:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ks := &defaultKnativeServing
			ingress := &defaultIngress
			vs := &defaultVirtualService

			// Set annotation and label for test
			vs.SetAnnotations(test.annotations)
			vs.SetLabels(test.labels)

			initObjs := []runtime.Object{ks, ingress, vs}

			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, ks)
			s.AddKnownTypes(configv1.SchemeGroupVersion, ingress)
			s.AddKnownTypes(v1alpha3.SchemeGroupVersion, vs)

			cl := fake.NewFakeClient(initObjs...)
			r := &ReconcileKnativeServing{client: cl, scheme: s}

			if _, err := r.Reconcile(defaultRequest); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// Check if VirtualService is deleted.
			refetched := &v1alpha3.VirtualService{}
			err := cl.Get(context.TODO(), types.NamespacedName{Name: "vsName", Namespace: "vsNamespace"}, refetched)
			if test.deleted {
				if !errors.IsNotFound(err) {
					t.Fatalf("get: (%v)", err)
				}
			}
			if !test.deleted {
				if err != nil {
					t.Fatalf("get: (%v)", err)
				}
			}
		})
	}
}

func TestCustomCertsConfigMap(t *testing.T) {
	ks := &v1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving",
			Namespace: "knative-serving",
		},
		Spec: v1alpha1.KnativeServingSpec{
			ControllerCustomCerts: v1alpha1.CustomCerts{
				Name: "test-cm",
				Type: "ConfigMap",
			},
		},
	}

	serviceCAAnnotations := map[string]string{"service.alpha.openshift.io/inject-cabundle": "true"}
	trustedCALabels := map[string]string{"config.openshift.io/inject-trusted-cabundle": "true"}

	tests := []struct {
		name    string
		in      []runtime.Object
		out     []*corev1.ConfigMap
		outCtrl *appsv1.Deployment
	}{{
		name: "plain field",
		out: []*corev1.ConfigMap{
			cm("test-cm", nil, nil, nil, ""),
			cm("test-cm-service-ca", nil, serviceCAAnnotations, nil, ""),
			cm("test-cm-trusted-ca", trustedCALabels, nil, nil, ""),
		},
	}, {
		name: "upgrade from 1.6.0",
		in: []runtime.Object{
			ctrl(""),
			cm("test-cm", nil, serviceCAAnnotations, map[string]string{"test": "foo"}, "1"),
		},
		out: []*corev1.ConfigMap{
			cm("test-cm", nil, nil, nil, "1"), // TODO: maybe we shouldn't stomp, retaining behavior from master though.
			cm("test-cm-service-ca", nil, serviceCAAnnotations, nil, ""),
			cm("test-cm-trusted-ca", trustedCALabels, nil, nil, ""),
		},
		outCtrl: ctrl("1"),
	}, {
		name: "just one secondary already filled",
		in: []runtime.Object{
			ctrl("2"),
			cm("test-cm", nil, serviceCAAnnotations, nil, "3"),
			cm("test-cm-service-ca", nil, serviceCAAnnotations, nil, ""),
			cm("test-cm-trusted-ca", trustedCALabels, nil, map[string]string{"trustedCA": "baz"}, ""),
		},
		out: []*corev1.ConfigMap{
			cm("test-cm", nil, nil, map[string]string{"trustedCA": "baz"}, "3"),
			cm("test-cm-service-ca", nil, serviceCAAnnotations, nil, ""),
			cm("test-cm-trusted-ca", trustedCALabels, nil, map[string]string{"trustedCA": "baz"}, ""),
		},
		outCtrl: ctrl("3"),
	}, {
		name: "both secondaries filled",
		in: []runtime.Object{
			ctrl("0"),
			cm("test-cm", nil, serviceCAAnnotations, nil, "1"),
			cm("test-cm-service-ca", nil, serviceCAAnnotations, map[string]string{"serviceCA": "bar"}, ""),
			cm("test-cm-trusted-ca", trustedCALabels, nil, map[string]string{"trustedCA": "baz"}, ""),
		},
		out: []*corev1.ConfigMap{
			cm("test-cm", nil, nil, map[string]string{"serviceCA": "bar", "trustedCA": "baz"}, "1"),
			cm("test-cm-service-ca", nil, serviceCAAnnotations, map[string]string{"serviceCA": "bar"}, ""),
			cm("test-cm-trusted-ca", trustedCALabels, nil, map[string]string{"trustedCA": "baz"}, ""),
		},
		outCtrl: ctrl("1"),
	}, {
		name: "certificate gets rolled",
		in: []runtime.Object{
			ctrl("10"),
			cm("test-cm", nil, serviceCAAnnotations, map[string]string{"serviceCA": "bar", "trustedCA": "baz"}, "100"),
			cm("test-cm-service-ca", nil, serviceCAAnnotations, map[string]string{"serviceCA": "bar"}, ""),
			cm("test-cm-trusted-ca", trustedCALabels, nil, map[string]string{"trustedCA": "baz2"}, ""),
		},
		out: []*corev1.ConfigMap{
			cm("test-cm", nil, nil, map[string]string{"serviceCA": "bar", "trustedCA": "baz2"}, "100"),
			cm("test-cm-service-ca", nil, serviceCAAnnotations, map[string]string{"serviceCA": "bar"}, ""),
			cm("test-cm-trusted-ca", trustedCALabels, nil, map[string]string{"trustedCA": "baz2"}, ""),
		},
		outCtrl: ctrl("100"),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cl := fake.NewFakeClient(test.in...)
			s := scheme.Scheme
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, ks)

			r := &ReconcileKnativeServing{client: cl, scheme: s}

			if err := r.ensureCustomCertsConfigMap(ks); err != nil {
				t.Fatal(err)
			}

			for _, want := range test.out {
				got := &corev1.ConfigMap{}
				if err := cl.Get(context.TODO(), types.NamespacedName{Name: want.Name, Namespace: want.Namespace}, got); err != nil {
					t.Fatalf("Failed to fetch cm: %v", err)
				}

				// Avoid ownerRef comparison for now.
				got.OwnerReferences = nil

				if !equality.Semantic.DeepEqual(got, want) {
					t.Fatalf("ConfigMaps %#v not equal to %#v", got, want)
				}
			}

			if test.outCtrl != nil {
				got := &appsv1.Deployment{}
				if err := cl.Get(context.TODO(), types.NamespacedName{Name: test.outCtrl.Name, Namespace: test.outCtrl.Namespace}, got); err != nil {
					t.Fatalf("Failed to fetch controller: %v", err)
				}

				if !equality.Semantic.DeepEqual(got, test.outCtrl) {
					t.Fatalf("ConfigMaps %#v not equal to %#v", got, test.outCtrl)
				}
			}
		})
	}
}

func ctrl(certVersion string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "knative-serving",
			Name:      "controller",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"serving.knative.openshift.io/mounted-cert-version": certVersion,
					},
				},
			},
		},
	}
}

func cm(name string, labels, annotations, data map[string]string, resourceVersion string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "knative-serving",
			Name:            name,
			Annotations:     annotations,
			Labels:          labels,
			ResourceVersion: resourceVersion,
		},
		Data: data,
	}
}
