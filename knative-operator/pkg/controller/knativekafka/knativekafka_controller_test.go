package knativekafka

import (
	"context"
	"os"
	"testing"
	"time"

	v1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	apis "knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	admissiontypes "sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

var (
	defaultRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: "knative-eventing", Name: "knative-kafka"},
	}
	deleteTime = metav1.NewTime(time.Now())
)

type fakeManager struct {
	client client.Client
	scheme *runtime.Scheme
}

func (*fakeManager) Add(manager.Runnable) error {
	return nil
}

func (*fakeManager) SetFields(interface{}) error {
	return nil
}

func (*fakeManager) Start(<-chan struct{}) error {
	return nil
}

func (*fakeManager) GetConfig() *rest.Config {
	return nil
}

func (f *fakeManager) GetScheme() *runtime.Scheme {
	return f.scheme
}

func (*fakeManager) GetAdmissionDecoder() admissiontypes.Decoder {
	return nil
}

func (f *fakeManager) GetClient() client.Client {
	return f.client
}

func (*fakeManager) GetFieldIndexer() client.FieldIndexer {
	return nil
}

func (*fakeManager) GetCache() cache.Cache {
	return nil
}

func (*fakeManager) GetRecorder(name string) record.EventRecorder {
	return nil
}

func (*fakeManager) GetRESTMapper() meta.RESTMapper {
	return nil
}

var _ manager.Manager = &fakeManager{}

func init() {
	os.Setenv("OPERATOR_NAME", "TEST_OPERATOR")
	os.Setenv("KAFKACHANNEL_MANIFEST_PATH", "testdata/kafkachannel-latest.yaml")
	os.Setenv("KAFKASOURCE_MANIFEST_PATH", "testdata/kafkasource-latest.yaml")
}

func TestKnativeKafkaReconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	tests := []struct {
		name     string
		instance v1alpha1.KnativeKafka
	}{
		{
			name: "Create CR with channel and source enabled",
			instance: v1alpha1.KnativeKafka{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "knative-kafka",
					Namespace:         "knative-eventing",
					DeletionTimestamp: nil,
				},
				Spec: v1alpha1.KnativeKafkaSpec{
					Source: v1alpha1.Source{
						Enabled: true,
					},
					Channel: v1alpha1.Channel{
						Enabled:          true,
						BootstrapServers: "foo.bar.com",
					},
				},
				Status: v1alpha1.KnativeKafkaStatus{
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
						},
					},
				},
			},
		},
		//{
		//	name: "Create CR with channel enabled and source disabled",
		//	instance: v1alpha1.KnativeKafka{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:      "knative-kafka",
		//			Namespace: "knative-eventing",
		//			DeletionTimestamp: nil,
		//		},
		//		Spec: v1alpha1.KnativeKafkaSpec{
		//			Source: v1alpha1.Source{
		//				Enabled: true,
		//			},
		//			Channel: v1alpha1.Channel{
		//				Enabled:          true,
		//				BootstrapServers: "foo.bar.com",
		//			},
		//		},
		//		Status: v1alpha1.KnativeKafkaStatus{
		//			Status: duckv1.Status{
		//				Conditions: []apis.Condition{
		//					{
		//						Status: "True",
		//						Type:   "DeploymentsAvailable",
		//					},
		//					{
		//						Status: "True",
		//						Type:   "InstallSucceeded",
		//					},
		//				},
		//			},
		//		},
		//	},
		//},
		//{
		//	name: "Create CR with channel disabled and source enabled",
		//	instance: v1alpha1.KnativeKafka{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:      "knative-kafka",
		//			Namespace: "knative-eventing",
		//			DeletionTimestamp: nil,
		//		},
		//		Spec: v1alpha1.KnativeKafkaSpec{
		//			Source: v1alpha1.Source{
		//				Enabled: true,
		//			},
		//			Channel: v1alpha1.Channel{
		//				Enabled:          true,
		//				BootstrapServers: "foo.bar.com",
		//			},
		//		},
		//		Status: v1alpha1.KnativeKafkaStatus{
		//			Status: duckv1.Status{
		//				Conditions: []apis.Condition{
		//					{
		//						Status: "True",
		//						Type:   "DeploymentsAvailable",
		//					},
		//					{
		//						Status: "True",
		//						Type:   "InstallSucceeded",
		//					},
		//				},
		//			},
		//		},
		//	},
		//},
		//{
		//	name: "Delete CR",
		//	instance: v1alpha1.KnativeKafka{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:      "knative-kafka",
		//			Namespace: "knative-eventing",
		//			DeletionTimestamp: &deleteTime,
		//		},
		//		Spec: v1alpha1.KnativeKafkaSpec{
		//			Source: v1alpha1.Source{
		//				Enabled: true,
		//			},
		//			Channel: v1alpha1.Channel{
		//				Enabled:          true,
		//				BootstrapServers: "foo.bar.com",
		//			},
		//		},
		//		Status: v1alpha1.KnativeKafkaStatus{
		//			Status: duckv1.Status{
		//				Conditions: []apis.Condition{
		//					{
		//						Status: "True",
		//						Type:   "DeploymentsAvailable",
		//					},
		//					{
		//						Status: "True",
		//						Type:   "InstallSucceeded",
		//					},
		//				},
		//			},
		//		},
		//	},
		//},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &test.instance)

			initObjs := []runtime.Object{&test.instance}

			cl := fake.NewFakeClient(initObjs...)
			mgr := &fakeManager{
				client: cl,
				scheme: s,
			}
			r, err := newReconciler(mgr)

			// Reconcile to intialize
			if _, err := r.Reconcile(defaultRequest); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// check if KafkaChannel controller deployment is created
			chDeploy := &appsv1.Deployment{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "kafka-ch-controller", Namespace: "knative-eventing"}, chDeploy)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}

			// check if KafkaSource controller deployment is created
			srcDeploy := &appsv1.Deployment{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "kafka-controller-manager", Namespace: "knative-sources"}, srcDeploy)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}

			// Delete KafkaChannel controller deployment.
			err = cl.Delete(context.TODO(), chDeploy)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}

			// Delete KafkaSource controller deployment.
			err = cl.Delete(context.TODO(), srcDeploy)
			if err != nil {
				t.Fatalf("delete: (%v)", err)
			}

			// Reconcile again
			if _, err := r.Reconcile(defaultRequest); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// Check again if KafkaChannel deployment is created after reconcile.
			chDeploy = &appsv1.Deployment{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "kafka-ch-controller", Namespace: "knative-eventing"}, chDeploy)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}

			// Check again if KafkaSource controller deployment is created after reconcile.
			srcDeploy = &appsv1.Deployment{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "kafka-controller-manager", Namespace: "knative-sources"}, srcDeploy)
			if err != nil {
				t.Fatalf("get: (%v)", err)
			}
		})
	}
}
