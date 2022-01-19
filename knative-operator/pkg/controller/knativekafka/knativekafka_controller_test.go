package knativekafka

import (
	"context"
	"fmt"
	"testing"
	"time"

	kafkaconfig "knative.dev/eventing-kafka/pkg/common/config"
	"sigs.k8s.io/yaml"

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
			cl := fake.NewClientBuilder().WithObjects(test.instance, &operatorv1alpha1.KnativeEventing{}).Build()

			kafkaChannelManifest, err := mf.ManifestFrom(mf.Path("testdata/channel/1-channel-consolidated.yaml"))
			if err != nil {
				t.Fatalf("failed to load KafkaChannel manifest: %v", err)
			}

			kafkaSourceManifest, err := mf.ManifestFrom(mf.Path("testdata/source/1-source.yaml"))
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
				err = cl.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("rbac-proxy-reviews-prom-rb-%s", deployment.Name)}, crb)
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

func TestUpdateEventingKafka(t *testing.T) {
	tests := []struct {
		name         string
		obj          *unstructured.Unstructured
		kafkaChannel v1alpha1.Channel
		expect       *unstructured.Unstructured
	}{{
		name: "Update config-kafka with all arguments",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
			},
		},
		kafkaChannel: v1alpha1.Channel{
			AuthSecretName:      "my-secret",
			BootstrapServers:    "example.com:1234",
			AuthSecretNamespace: "my-ns",
		},
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
				"data": map[string]interface{}{
					"eventing-kafka": "kafka:\n  authSecretName: my-secret\n  authSecretNamespace: my-ns\n  brokers: example.com:1234\n",
				},
			},
		},
	}, {
		name: "Update config-kafka with only brokers",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
			},
		},
		kafkaChannel: v1alpha1.Channel{
			BootstrapServers: "example.com:1234",
		},
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
				"data": map[string]interface{}{
					"eventing-kafka": "kafka:\n  brokers: example.com:1234\n",
				},
			},
		},
	}, {
		name: "Update config-kafka - overwrite all arguments",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
				"data": map[string]interface{}{
					"eventing-kafka": marshalEventingKafkaConfig(EventingKafkaConfig{
						Kafka: kafkaconfig.EKKafkaConfig{
							Brokers:             "TO_BE_OVERWRITTEN",
							AuthSecretName:      "TO_BE_OVERWRITTEN",
							AuthSecretNamespace: "TO_BE_OVERWRITTEN",
						},
					}),
				},
			},
		},
		kafkaChannel: v1alpha1.Channel{
			BootstrapServers:    "example.com:1234",
			AuthSecretName:      "my-secret",
			AuthSecretNamespace: "my-ns",
		},
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
				"data": map[string]interface{}{
					"eventing-kafka": "kafka:\n  authSecretName: my-secret\n  authSecretNamespace: my-ns\n  brokers: example.com:1234\n",
				},
			},
		},
	}, {
		name: "Update config-kafka - overwrite only brokers",
		obj: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
				"data": map[string]interface{}{
					"eventing-kafka": marshalEventingKafkaConfig(EventingKafkaConfig{
						Kafka: kafkaconfig.EKKafkaConfig{
							Brokers: "TO_BE_OVERWRITTEN",
						},
					}),
				},
			},
		},
		kafkaChannel: v1alpha1.Channel{
			BootstrapServers: "example.com:1234",
		},
		expect: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "config-kafka",
				},
				"data": map[string]interface{}{
					"eventing-kafka": "kafka:\n  brokers: example.com:1234\n",
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
		kafkaChannel: v1alpha1.Channel{
			AuthSecretName:      "my-secret",
			AuthSecretNamespace: "my-ns",
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
		kafkaChannel: v1alpha1.Channel{
			AuthSecretName:      "my-secret",
			AuthSecretNamespace: "my-ns",
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
			err := configureLegacyEventingKafka(test.kafkaChannel)(test.obj)
			if err != nil {
				t.Fatalf("setAuthSecretNamespace/setAuthSecretName: (%v)", err)
			}

			if !cmp.Equal(test.expect, test.obj) {
				t.Fatalf("Resource wasn't what we expected, diff: %s", cmp.Diff(test.obj, test.expect))
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

func TestDisabledControllers(t *testing.T) {
	tests := []struct {
		name         string
		knativeKafka v1alpha1.KnativeKafkaSpec
		expect       *unstructured.Unstructured
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
		expect: makeEventingKafkaDeployment(t, "sink-controller"),
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
		expect: makeEventingKafkaDeployment(t, "broker-controller,trigger-controller"),
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
		expect: makeEventingKafkaDeployment(t, ""),
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
		expect: makeEventingKafkaDeployment(t, "broker-controller,trigger-controller,sink-controller"),
	}}

	for _, test := range tests {
		// by default we assume all disabled:
		defaultDeployment := makeEventingKafkaDeployment(t, "broker-controller,trigger-controller,sink-controller")
		t.Run(test.name, func(t *testing.T) {
			err := configureEventingKafka(test.knativeKafka)(defaultDeployment)
			if err != nil {
				t.Fatalf("configureKafkaBroker: (%v)", err)
			}

			if !cmp.Equal(test.expect, defaultDeployment) {
				t.Fatalf("Resource wasn't what we expected, diff: %s", cmp.Diff(defaultDeployment, test.expect))
			}
		})
	}
}

func makeEventingKafkaDeployment(t *testing.T, disabledControllers string) *unstructured.Unstructured {
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
	d.Spec.Template.Spec.Containers[0].Args = []string{"--disable-controllers=" + disabledControllers}

	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(d, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured Deployment: %v, err: %v", d, err)
	}

	return result

}

func marshalEventingKafkaConfig(kafka EventingKafkaConfig) string {
	configBytes, _ := yaml.Marshal(kafka)
	return string(configBytes)
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
			HighAvailability: &operatorv1alpha1.HighAvailability{
				Replicas: 1,
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
		name:           "kafka channel controller",
		deploymentName: "kafka-ch-controller",
		shouldFail:     false,
	}, {
		name:           "kafka webhook",
		deploymentName: "kafka-webhook",
		shouldFail:     true,
	}, {
		name:           "kafka source controller",
		deploymentName: "kafka-controller-manager",
		shouldFail:     false,
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
