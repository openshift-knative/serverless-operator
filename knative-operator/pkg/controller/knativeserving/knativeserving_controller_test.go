package knativeserving

import (
	"context"
	"os"
	"testing"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	"k8s.io/apimachinery/pkg/api/equality"

	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	apis "knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/dashboard"
)

var (
	defaultKnativeServing = v1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving",
			Namespace: "knative-serving",
		},
		Status: v1alpha1.KnativeServingStatus{
			Status: duckv1.Status{
				Conditions: []apis.Condition{
					{
						Status: "True",
						Type:   "DeploymentsAvailable",
					},
					{
						Status: "True",
						Type:   "InstallSucceeded",
					},
					{
						Status: "True",
						Type:   "VersionMigrationEligible",
					},
				},
			},
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

	defaultRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: "knative-serving", Name: "knative-serving"},
	}

	defaultKnService = servingv1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: servingv1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kn-cli",
			Namespace: "knative-serving",
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "kn-download-server",
									Image: "fake.example.com/openshift/kn-cli-artifacts:latest",
								},
							},
						},
					},
				},
			},
		},
		Status: servingv1.ServiceStatus{
			Status: duckv1.Status{
				Conditions: []apis.Condition{
					{
						Status: "True",
						Type:   "Ready",
					},
				},
			},
			RouteStatusFields: servingv1.RouteStatusFields{
				URL: &apis.URL{Host: "kn-cli-knative-serving.example.com"},
			},
		},
	}

	dashboardNamespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: dashboard.ConfigManagedNamespace,
		},
	}
)

func init() {
	os.Setenv("OPERATOR_NAME", "TEST_OPERATOR")
	os.Setenv("KOURIER_MANIFEST_PATH", "kourier/testdata/kourier-latest.yaml")
	os.Setenv(dashboard.ServingDashboardPathEnvVar, "../dashboard/testdata/grafana-dash-knative.yaml")
	os.Setenv("CONSOLECLIDOWNLOAD_MANIFEST_PATH", "consoleclidownload/testdata/console_cli_download_kn_resources.yaml")
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
			ccd := &consolev1.ConsoleCLIDownload{}
			ns := &dashboardNamespace
			knService := &defaultKnService
			monitor := &monitoringv1.ServiceMonitor{}
			initObjs := []runtime.Object{ks, ingress, ns, knService}

			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, ks)
			s.AddKnownTypes(configv1.SchemeGroupVersion, ingress)
			s.AddKnownTypes(consolev1.GroupVersion, ccd)
			s.AddKnownTypes(servingv1.SchemeGroupVersion, knService)
			s.AddKnownTypes(routev1.GroupVersion, &routev1.Route{})
			scheme.Scheme.AddKnownTypes(monitoringv1.SchemeGroupVersion, monitor)
			cl := fake.NewFakeClient(initObjs...)
			r := &ReconcileKnativeServing{client: cl, scheme: s}

			// Reconcile to initialize
			if _, err := r.Reconcile(defaultRequest); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// Check if Kourier is deployed.
			deploy := &appsv1.Deployment{}
			err := cl.Get(context.TODO(), types.NamespacedName{Name: "3scale-kourier-gateway", Namespace: "knative-serving-ingress"}, deploy)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}

			// Check kn ConsoleCLIDownload CR
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "kn", Namespace: ""}, ccd)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}

			// Check if Serving dashboard configmap is available
			dashboardCM := &corev1.ConfigMap{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative", Namespace: ns.Name}, dashboardCM)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}

			// Delete Kourier deployment.
			err = cl.Delete(context.TODO(), deploy)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}

			// Delete ConsoleCLIDownload CR
			err = cl.Delete(context.TODO(), ccd)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}

			// Delete Dashboard configmap.
			err = cl.Delete(context.TODO(), dashboardCM)
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

			// Check again if Kourier deployment is created after reconcile.
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "3scale-kourier-gateway", Namespace: "knative-serving-ingress"}, deploy)
			checkError(t, err)

			// Check again if Serving dashboard configmap is available.
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "grafana-dashboard-definition-knative", Namespace: ns.Name}, dashboardCM)
			checkError(t, err)
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

	serviceCAAnnotations := map[string]string{serviceCAKey: "true"}
	trustedCALabels := map[string]string{trustedCAKey: "true"}

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

// TestKnativeServingStatus tests KnativeServing CR status with Kourier's installation failure.
func TestKnativeServingStatus(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	ks := &defaultKnativeServing
	ingress := &defaultIngress
	knService := &defaultKnService

	initObjs := []runtime.Object{ks, ingress, knService}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, ks)
	s.AddKnownTypes(configv1.SchemeGroupVersion, ingress)
	s.AddKnownTypes(servingv1.SchemeGroupVersion, knService)

	cl := fake.NewFakeClient(initObjs...)
	r := &ReconcileKnativeServing{client: cl, scheme: s}

	// Test with invalid Kourier manifest file.
	os.Setenv("KOURIER_MANIFEST_PATH", "kourier/testdata/non-exist-file")
	if _, err := r.Reconcile(defaultRequest); err == nil {
		t.Fatalf("reconcile does not fail with invalid manifest path")
	}

	failedKs := &v1alpha1.KnativeServing{}
	err := cl.Get(context.TODO(), types.NamespacedName{Name: "knative-serving", Namespace: "knative-serving"}, failedKs)
	if err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if failedKs.Status.GetCondition(v1alpha1.DependenciesInstalled).Status != corev1.ConditionFalse {
		t.Fatalf("status: (%v)", failedKs.Status.GetCondition(v1alpha1.DependenciesInstalled))
	}

	// Reconcile with correct Kourier manifest file.
	os.Setenv("KOURIER_MANIFEST_PATH", "kourier/testdata/kourier-latest.yaml")
	if _, err := r.Reconcile(defaultRequest); err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	successKs := &v1alpha1.KnativeServing{}
	err = cl.Get(context.TODO(), types.NamespacedName{Name: "knative-serving", Namespace: "knative-serving"}, successKs)
	if err != nil {
		t.Fatalf("get: (%v)", err)
	}
	if successKs.Status.GetCondition(v1alpha1.DependenciesInstalled).Status != corev1.ConditionTrue {
		t.Fatalf("status: (%v)", failedKs.Status.GetCondition(v1alpha1.DependenciesInstalled))
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
						certVersionKey: certVersion,
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
