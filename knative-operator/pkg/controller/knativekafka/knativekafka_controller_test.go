package knativekafka

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring"
)

var (
	defaultRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: "knative-eventing", Name: "knative-kafka"},
	}
)

func init() {
	apis.AddToScheme(scheme.Scheme)
}

func TestKnativeKafkaReconcile(t *testing.T) {
	tests := []struct {
		name         string
		instance     *v1alpha1.KnativeKafka
		exists       []types.NamespacedName
		doesNotExist []types.NamespacedName
	}{{
		name:     "Create CR with channel and source enabled",
		instance: makeCr(withChannelEnabled, withSourceEnabled),
		exists: []types.NamespacedName{
			{Name: "kafka-channel-dispatcher", Namespace: "knative-eventing"},
			{Name: "kafka-channel-receiver", Namespace: "knative-eventing"},
			{Name: "kafka-source-dispatcher", Namespace: "knative-eventing"},
		},
		doesNotExist: []types.NamespacedName{},
	}, {
		name:     "Create CR with channel enabled and source disabled",
		instance: makeCr(withChannelEnabled),
		exists: []types.NamespacedName{
			{Name: "kafka-channel-dispatcher", Namespace: "knative-eventing"},
			{Name: "kafka-channel-receiver", Namespace: "knative-eventing"},
		},
		doesNotExist: []types.NamespacedName{
			{Name: "kafka-source-dispatcher", Namespace: "knative-eventing"},
		},
	}, {
		name:     "Create CR with channel disabled and source enabled",
		instance: makeCr(withSourceEnabled),
		exists: []types.NamespacedName{
			{Name: "kafka-source-dispatcher", Namespace: "knative-eventing"},
		},
		doesNotExist: []types.NamespacedName{
			{Name: "kafka-channel-dispatcher", Namespace: "knative-eventing"},
			{Name: "kafka-channel-receiver", Namespace: "knative-eventing"},
		},
	}, {
		name:     "Create CR with channel and source disabled",
		instance: makeCr(),
		exists:   []types.NamespacedName{},
		doesNotExist: []types.NamespacedName{
			{Name: "kafka-channel-dispatcher", Namespace: "knative-eventing"},
			{Name: "kafka-channel-receiver", Namespace: "knative-eventing"},
			{Name: "kafka-source-dispatcher", Namespace: "knative-eventing"},
		},
	}, {
		name:     "Delete CR",
		instance: makeCr(withChannelEnabled, withSourceEnabled, withDeleted),
		exists:   []types.NamespacedName{},
		doesNotExist: []types.NamespacedName{
			{Name: "kafka-ch-controller", Namespace: "knative-eventing"},
			{Name: "kafka-source-dispatcher", Namespace: "knative-eventing"},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cl := fake.NewClientBuilder().WithObjects(test.instance, &operatorv1alpha1.KnativeEventing{}).Build()

			kafkaChannelManifest, err := mf.ManifestFrom(mf.Path("testdata/channel/eventing-kafka-channel.yaml"))
			if err != nil {
				t.Fatalf("failed to load KafkaChannel manifest: %v", err)
			}

			kafkaSourceManifest, err := mf.ManifestFrom(mf.Path("testdata/source/eventing-kafka-source.yaml"))
			if err != nil {
				t.Fatalf("failed to load KafkaSource manifest: %v", err)
			}

			r := &ReconcileKnativeKafka{
				client:                  cl,
				scheme:                  scheme.Scheme,
				rawKafkaChannelManifest: kafkaChannelManifest,
				rawKafkaSourceManifest:  kafkaSourceManifest,
			}

			// Reconcile to initialize
			if _, err := r.Reconcile(context.Background(), defaultRequest); err != nil {
				t.Fatalf("reconcile: (%v)", err)
			}

			// check if things that should exist is created
			for _, d := range test.exists {
				deployment := &appsv1.Deployment{}
				err := cl.Get(context.TODO(), d, deployment)
				if err != nil {
					t.Fatalf("get: (%v)", err)
				}
				// Check if rbac proxy is injected
				if len(deployment.Spec.Template.Spec.Containers) != 2 {
					t.Fatal("rbac proxy not injected")
				}

				// Check if the service monitor for the Kafka deployment is created
				sm := &monitoringv1.ServiceMonitor{}
				err = cl.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-sm", deployment.Name), Namespace: "knative-eventing"}, sm)
				if err != nil {
					t.Fatalf("get: (%v)", err)
				}

				// Check if the service monitor service for the Kafka deployment is created
				sms := &corev1.Service{}
				err = cl.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-sm-service", deployment.Name), Namespace: "knative-eventing"}, sms)
				if err != nil {
					t.Fatalf("get: (%v)", err)
				}

				// Check if the clusterrolebinding for the Kafka deployment is created
				crb := &rbacv1.ClusterRoleBinding{}
				name := monitoring.IndexByName[deployment.Name]
				err = cl.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("rbac-proxy-reviews-prom-rb-%s", name.ServiceAccountName)}, crb)
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
			if _, err := r.Reconcile(context.Background(), defaultRequest); err != nil {
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

func TestBrokerCfg(t *testing.T) {
	tests := []struct {
		name         string
		obj          *unstructured.Unstructured
		knativeKafka v1alpha1.KnativeKafkaSpec
		expect       *unstructured.Unstructured
	}{{
		name: "Update kafka-broker-config with all arguments",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "kafka-broker-config",
				},
			},
		},
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Broker: v1alpha1.Broker{
				DefaultConfig: v1alpha1.BrokerDefaultConfig{
					AuthSecretName:    "my-secret",
					NumPartitions:     12,
					ReplicationFactor: 3,
					BootstrapServers:  "example.com:1234",
				},
			},
		},
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "kafka-broker-config",
				},
				"data": map[string]interface{}{
					"bootstrap.servers":                "example.com:1234",
					"auth.secret.ref.name":             "my-secret",
					"default.topic.partitions":         "12",
					"default.topic.replication.factor": "3",
				},
			},
		},
	}, {
		name: "Update kafka-broker-config with bootstrap and topic settings",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "kafka-broker-config",
				},
			},
		},
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Broker: v1alpha1.Broker{
				DefaultConfig: v1alpha1.BrokerDefaultConfig{
					NumPartitions:     12,
					ReplicationFactor: 3,
					BootstrapServers:  "example.com:1234",
				},
			},
		},
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "kafka-broker-config",
				},
				"data": map[string]interface{}{
					"bootstrap.servers":                "example.com:1234",
					"default.topic.partitions":         "12",
					"default.topic.replication.factor": "3",
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
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Broker: v1alpha1.Broker{
				DefaultConfig: v1alpha1.BrokerDefaultConfig{
					AuthSecretName:    "my-secret",
					NumPartitions:     12,
					ReplicationFactor: 3,
					BootstrapServers:  "example.com:1234",
				},
			},
		},
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
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Broker: v1alpha1.Broker{
				DefaultConfig: v1alpha1.BrokerDefaultConfig{
					AuthSecretName:    "my-secret",
					NumPartitions:     12,
					ReplicationFactor: 3,
					BootstrapServers:  "example.com:1234",
				},
			},
		},
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
			err := configureEventingKafka(test.knativeKafka)(test.obj)
			if err != nil {
				t.Fatalf("configureKafkaBroker: (%v)", err)
			}

			if !cmp.Equal(test.expect, test.obj) {
				t.Fatalf("Resource wasn't what we expected, diff: %s", cmp.Diff(test.obj, test.expect))
			}
		})
	}
}

func TestChannelCfg(t *testing.T) {
	tests := []struct {
		name         string
		obj          *unstructured.Unstructured
		knativeKafka v1alpha1.KnativeKafkaSpec
		expect       *unstructured.Unstructured
	}{{
		name: "Update kafka-channel-config with all arguments",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "kafka-channel-config",
				},
			},
		},
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Channel: v1alpha1.Channel{
				AuthSecretName:      "my-secret",
				AuthSecretNamespace: "my-secret-ns",
				BootstrapServers:    "example.com:1234",
			},
		},
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "kafka-channel-config",
				},
				"data": map[string]interface{}{
					"bootstrap.servers":         "example.com:1234",
					"auth.secret.ref.name":      "my-secret",
					"auth.secret.ref.namespace": "my-secret-ns",
				},
			},
		},
	}, {
		name: "Update kafka-broker-config with bootstrap server setting only",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "kafka-channel-config",
				},
			},
		},
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Channel: v1alpha1.Channel{
				BootstrapServers: "example.com:1234",
			},
		},
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "kafka-channel-config",
				},
				"data": map[string]interface{}{
					"bootstrap.servers": "example.com:1234",
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
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Channel: v1alpha1.Channel{
				BootstrapServers:    "example.com:1234",
				AuthSecretName:      "my-secret",
				AuthSecretNamespace: "my-secret-ns",
			},
		},
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
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Channel: v1alpha1.Channel{
				AuthSecretName:      "my-secret",
				AuthSecretNamespace: "my-secret-ns",
				BootstrapServers:    "example.com:1234",
			},
		},
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
			err := configureEventingKafka(test.knativeKafka)(test.obj)
			if err != nil {
				t.Fatalf("configureKafkaChannel: (%v)", err)
			}

			if !cmp.Equal(test.expect, test.obj) {
				t.Fatalf("Resource wasn't what we expected, diff: %s", cmp.Diff(test.obj, test.expect))
			}
		})
	}
}

func TestDisabledControllers(t *testing.T) {
	tests := []struct {
		name                        string
		knativeKafka                v1alpha1.KnativeKafkaSpec
		expectedDisabledControllers []string
	}{{
		name: "just broker",
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Broker: v1alpha1.Broker{
				Enabled: true,
			},
			Sink: v1alpha1.Sink{
				Enabled: false,
			},
		},
		expectedDisabledControllers: []string{"sink-controller", "source-controller"},
	}, {
		name: "just sink",
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Broker: v1alpha1.Broker{
				Enabled: false,
			},
			Sink: v1alpha1.Sink{
				Enabled: true,
			},
		},
		expectedDisabledControllers: []string{"broker-controller", "trigger-controller", "source-controller"},
	}, {
		name: "broker and sink",
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Broker: v1alpha1.Broker{
				Enabled: true,
			},
			Sink: v1alpha1.Sink{
				Enabled: true,
			},
		},
		expectedDisabledControllers: []string{"source-controller"},
	}, {
		name: "no broker and no sink",
		knativeKafka: v1alpha1.KnativeKafkaSpec{
			Broker: v1alpha1.Broker{
				Enabled: false,
			},
			Sink: v1alpha1.Sink{
				Enabled: false,
			},
		},
		expectedDisabledControllers: []string{"broker-controller", "trigger-controller", "sink-controller", "source-controller"},
	}}

	for _, test := range tests {
		defaultDeployment := makeEventingKafkaDeployment(t) //, "")
		t.Run(test.name, func(t *testing.T) {
			err := configureEventingKafka(test.knativeKafka)(defaultDeployment)
			if err != nil {
				t.Fatalf("configureKafkaBroker: (%v)", err)
			}

			// disabled controller arguments are stored on first container, as first arguemt
			disabledControllerArgs := extractDeployment(t, defaultDeployment).Spec.Template.Spec.Containers[0].Args[0]
			for _, v := range test.expectedDisabledControllers {
				assert.True(t, strings.Contains(disabledControllerArgs, v))
			}
		})
	}
}

func extractDeployment(t *testing.T, resource *unstructured.Unstructured) *appsv1.Deployment {
	var deployment = &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(resource, deployment, nil); err != nil {
		t.Fatalf("Could not create Deployment: %v, err: %v", resource, err)
	}
	return deployment
}

func makeEventingKafkaDeployment(t *testing.T) *unstructured.Unstructured {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kafka-controller",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "controller",
						},
					},
				},
			},
		},
	}

	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(d, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured Deployment: %v, err: %v", d, err)
	}

	return result
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
			HighAvailability: &base.HighAvailability{
				Replicas: ptr.Int32(1),
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

func TestCheckHAComponent(t *testing.T) {
	cases := []struct {
		name           string
		deploymentName string
		shouldFail     bool
	}{{
		name:           "Eventing Kafka Controller",
		deploymentName: "kafka-controller",
		shouldFail:     false,
	}, {
		name:           "Eventing Kafka Webhook",
		deploymentName: "kafka-webhook-eventing",
		shouldFail:     false,
	}, {
		name:           "kafka webhook",
		deploymentName: "kafka-webhook",
		shouldFail:     true,
	}, {
		name:           "kafka channel dispatcher",
		deploymentName: "kafka-ch-dispatcher",
		shouldFail:     true,
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checkHAComponent(tc.deploymentName)
			if result == tc.shouldFail {
				t.Errorf("Got: %v, want: %v\n", result, tc.shouldFail)
			}
		})
	}
}

func TestMonitoringResources(t *testing.T) {

	kk := &v1alpha1.KnativeKafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-kafka",
			Namespace: "knative-eventing",
		},
		Spec: v1alpha1.KnativeKafkaSpec{
			Broker:  v1alpha1.Broker{Enabled: true},
			Source:  v1alpha1.Source{Enabled: false},
			Sink:    v1alpha1.Sink{Enabled: true},
			Channel: v1alpha1.Channel{Enabled: false},
		},
	}

	t.Setenv("KAFKACHANNEL_MANIFEST_PATH", "../../../deploy/resources/knativekafka/channel")
	t.Setenv("KAFKASOURCE_MANIFEST_PATH", "../../../deploy/resources/knativekafka/source")
	t.Setenv("KAFKACONTROLLER_MANIFEST_PATH", "../../../deploy/resources/knativekafka/controller")
	t.Setenv("KAFKABROKER_MANIFEST_PATH", "../../../deploy/resources/knativekafka/broker")
	t.Setenv("KAFKASINK_MANIFEST_PATH", "../../../deploy/resources/knativekafka/sink")

	r, err := newReconciler(&MockManager{})
	if err != nil {
		t.Fatal(err)
	}

	manifests, err := r.buildManifest(kk, manifestBuildEnabledOnly)
	if err != nil {
		t.Fatal(err)
	}

	crGvk := schema.GroupVersionKind{
		Group:   rbacv1.SchemeGroupVersion.Group,
		Version: rbacv1.SchemeGroupVersion.Version,
		Kind:    "ClusterRoleBinding",
	}
	svGvk := schema.GroupVersionKind{
		Group:   monitoringv1.SchemeGroupVersion.Group,
		Version: monitoringv1.SchemeGroupVersion.Version,
		Kind:    "ServiceMonitor",
	}
	svcGvk := schema.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Service",
	}

	components := []monitoring.Component{
		monitoring.KafkaController,
		monitoring.KafkaWebhook,
		monitoring.KafkaBrokerReceiver,
		monitoring.KafkaBrokerDispatcher,
		monitoring.KafkaSinkReceiver,
	}
	svcs := sets.NewString()
	sMon := sets.NewString()

	for _, c := range components {
		svcs.Insert(c.Name + "-sm-service")
		sMon.Insert(c.Name + "-sm")
	}

	expected := map[schema.GroupVersionKind]sets.String{
		crGvk: sets.NewString(
			"rbac-proxy-reviews-prom-rb-kafka-controller",
			"rbac-proxy-reviews-prom-rb-kafka-webhook-eventing",
			"rbac-proxy-reviews-prom-rb-knative-kafka-broker-data-plane",
			"rbac-proxy-reviews-prom-rb-knative-kafka-sink-data-plane",
		),
		svGvk:  sMon,
		svcGvk: svcs,
	}

	for _, r := range manifests.Resources() {
		if expected, ok := expected[r.GroupVersionKind()]; ok {
			if !expected.Has(r.GetName()) {
				t.Log(r)
			}
			expected.Delete(r.GetName())
		}
	}

	for k, v := range expected {
		if v.Len() > 0 {
			t.Errorf("failed to find %+v, missing %v", k, v.List())
		}
	}
}

type MockManager struct {
	manager.Manager
}

func (m *MockManager) GetClient() client.Client {
	return nil
}

func (m *MockManager) GetScheme() *runtime.Scheme {
	return nil
}
