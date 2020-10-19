package knativekafka

import (
	"context"
	"testing"
	"time"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	defaultRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: "knative-eventing", Name: "knative-kafka"},
	}
)

func TestKnativeKafkaReconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	tests := []struct {
		name         string
		instance     *v1alpha1.KnativeKafka
		exists       []types.NamespacedName
		doesNotExist []types.NamespacedName
	}{{
		name:     "Create CR with channel and source enabled",
		instance: makeCr(withChannelEnabled, withSourceEnabled),
		exists: []types.NamespacedName{
			{Name: "kafka-ch-controller", Namespace: "knative-eventing"},
			{Name: "kafka-controller-manager", Namespace: "knative-eventing"},
		},
		doesNotExist: []types.NamespacedName{},
	}, {
		name:     "Create CR with channel enabled and source disabled",
		instance: makeCr(withChannelEnabled),
		exists: []types.NamespacedName{
			{Name: "kafka-ch-controller", Namespace: "knative-eventing"},
		},
		doesNotExist: []types.NamespacedName{
			{Name: "kafka-controller-manager", Namespace: "knative-eventing"},
		},
	}, {
		name:     "Create CR with channel disabled and source enabled",
		instance: makeCr(withSourceEnabled),
		exists: []types.NamespacedName{
			{Name: "kafka-controller-manager", Namespace: "knative-eventing"},
		},
		doesNotExist: []types.NamespacedName{
			{Name: "kafka-ch-controller", Namespace: "knative-eventing"},
		},
	}, {
		name:     "Create CR with channel and source disabled",
		instance: makeCr(),
		exists:   []types.NamespacedName{},
		doesNotExist: []types.NamespacedName{
			{Name: "kafka-ch-controller", Namespace: "knative-eventing"},
			{Name: "kafka-controller-manager", Namespace: "knative-eventing"},
		},
	}, {
		name:     "Delete CR",
		instance: makeCr(withChannelEnabled, withSourceEnabled, withDeleted),
		exists:   []types.NamespacedName{},
		doesNotExist: []types.NamespacedName{
			{Name: "kafka-ch-controller", Namespace: "knative-eventing"},
			{Name: "kafka-controller-manager", Namespace: "knative-eventing"},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, test.instance)

			initObjs := []runtime.Object{test.instance}

			cl := fake.NewFakeClient(initObjs...)

			kafkaChannelManifest, err := mf.ManifestFrom(mf.Path("testdata/kafkachannel-latest.yaml"))
			if err != nil {
				t.Fatalf("failed to load KafkaChannel manifest: %v", err)
			}

			kafkaSourceManifest, err := mf.ManifestFrom(mf.Path("testdata/kafkasource-latest.yaml"))
			if err != nil {
				t.Fatalf("failed to load KafkaSource manifest: %v", err)
			}

			r := &ReconcileKnativeKafka{
				client:                  cl,
				scheme:                  s,
				rawKafkaChannelManifest: kafkaChannelManifest,
				rawKafkaSourceManifest:  kafkaSourceManifest,
			}

			// Reconcile to intialize
			if _, err := r.Reconcile(defaultRequest); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// check if things that should exist is created
			for _, d := range test.exists {
				deployment := &appsv1.Deployment{}
				err := cl.Get(context.TODO(), d, deployment)
				if err != nil {
					t.Fatalf("get: (%v)", err)
				}
			}

			// check if things that shouldnt exist is deleted
			for _, d := range test.doesNotExist {
				deployment := &appsv1.Deployment{}
				err = cl.Get(context.TODO(), d, deployment)
				if err == nil || !errors.IsNotFound(err) {
					t.Fatalf("exists: (%v)", err)
				}
			}

			// delete deployments to see if they're recreated
			for _, d := range test.exists {
				deployment := &appsv1.Deployment{}
				err = cl.Get(context.TODO(), d, deployment)
				if err != nil {
					t.Fatalf("get: (%v)", err)
				}
				err = cl.Delete(context.TODO(), deployment)
				if err != nil {
					t.Fatalf("delete: (%v)", err)
				}
			}

			// Reconcile again
			if _, err := r.Reconcile(defaultRequest); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// check if things that should exist is created
			for _, d := range test.exists {
				deployment := &appsv1.Deployment{}
				err = cl.Get(context.TODO(), d, deployment)
				if err != nil {
					t.Fatalf("get: (%v)", err)
				}
			}
		})
	}
}

func TestSetBootstrapServers(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	tests := []struct {
		name             string
		obj              *unstructured.Unstructured
		bootstrapServers string
		expect           *unstructured.Unstructured
	}{{
		name: "Update config-kafka",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
			},
		},
		bootstrapServers: "example.com:1234",
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
				"data": map[string]interface{}{
					"bootstrapServers": "example.com:1234",
				},
			},
		},
	}, {
		name: "Update config-kafka - overwrite",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
				"data": map[string]interface{}{
					"bootstrapServers": "TO_BE_OVERWRITTEN",
				},
			},
		},
		bootstrapServers: "example.com:1234",
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
				"data": map[string]interface{}{
					"bootstrapServers": "example.com:1234",
				},
			},
		},
	}, {
		name: "Do not update other configmaps",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-foo",
				},
			},
		},
		bootstrapServers: "example.com:1234",
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-foo",
				},
			},
		},
	}, {
		name: "Do not update other resources",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
			},
		},
		bootstrapServers: "example.com:1234",
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := setBootstrapServers(test.bootstrapServers)(test.obj)
			if err != nil {
				t.Fatalf("setBootstrapServers: (%v)", err)
			}

			if !equality.Semantic.DeepEqual(test.expect, test.obj) {
				t.Fatalf("Resource wasn't what we expected: %#v, want %#v", test.obj, test.expect)
			}
		})
	}
}

func makeCr(mods ...func(*v1alpha1.KnativeKafka)) *v1alpha1.KnativeKafka {
	base := &v1alpha1.KnativeKafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "knative-kafka",
			Namespace:         "knative-eventing",
			DeletionTimestamp: nil,
		},
		Spec: v1alpha1.KnativeKafkaSpec{
			Source: v1alpha1.Source{
				Enabled: false,
			},
			Channel: v1alpha1.Channel{
				Enabled:          false,
				BootstrapServers: "foo.bar.com",
			},
		},
	}
	for _, mod := range mods {
		mod(base)
	}
	return base
}

func withSourceEnabled(kk *v1alpha1.KnativeKafka) {
	kk.Spec.Source.Enabled = true
}

func withChannelEnabled(kk *v1alpha1.KnativeKafka) {
	kk.Spec.Channel.Enabled = true
}

func withDeleted(kk *v1alpha1.KnativeKafka) {
	t := metav1.NewTime(time.Now())
	kk.ObjectMeta.DeletionTimestamp = &t
}
