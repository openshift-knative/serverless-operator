package monitoring

import (
	"fmt"
	serverlessoperatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"knative.dev/operator/pkg/apis/operator/base"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/yaml"
	"time"
)

func ExampleReconcileMonitoringForNamespacedBroker() {
	kk := &serverlessoperatorv1alpha1.KnativeKafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operator.serverless.openshift.io/v1alpha1",
			Kind:       "KnativeKafka",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "foo",
			Namespace:       "bar",
			UID:             "deadbeef",
			ResourceVersion: "123",
			Generation:      456,
			CreationTimestamp: metav1.Time{
				Time: time.UnixMilli(1234567890123),
			},
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{
				"baz": "fox",
			},
		},
		Spec: serverlessoperatorv1alpha1.KnativeKafkaSpec{
			Broker: serverlessoperatorv1alpha1.Broker{
				Enabled: true,
				DefaultConfig: serverlessoperatorv1alpha1.BrokerDefaultConfig{
					BootstrapServers:  "example.com",
					NumPartitions:     250,
					ReplicationFactor: 150,
					AuthSecretName:    "secret-name",
				},
			},
			Source: serverlessoperatorv1alpha1.Source{
				Enabled: false,
			},
			Sink: serverlessoperatorv1alpha1.Sink{
				Enabled: false,
			},
			Channel: serverlessoperatorv1alpha1.Channel{
				Enabled:             false,
				BootstrapServers:    "example.com",
				AuthSecretNamespace: "secret-ns",
				AuthSecretName:      "secret-name",
			},
			Config: map[string]map[string]string{
				"other-config": {
					"resources": "to be preserved",
				},
				"namespaced-broker-resources": {
					"resources": "to be overridden",
				},
			},
			HighAvailability: &base.HighAvailability{
				Replicas: pointer.Int32(25),
			},
			Logging: &serverlessoperatorv1alpha1.Logging{
				Level: "DEBUG",
			},
			Workloads: []base.WorkloadOverride{{
				Name: "WorkloadOverride",
			}},
		},
		Status: serverlessoperatorv1alpha1.KnativeKafkaStatus{
			Status: duckv1.Status{
				ObservedGeneration: 1234,
			},
			Version: "5678",
		},
	}
	err := ReconcileMonitoringForNamespacedBroker(kk)
	if err != nil {
		fmt.Printf("ReconcileMonitoringForNamespacedBroker() error: %v", err)
	}

	bytes, err := yaml.Marshal(kk)
	if err != nil {
		fmt.Printf("ReconcileMonitoringForNamespacedBroker() error marshaling output: %v", err)
	}

	fmt.Println(string(bytes))
	// Output:
	// apiVersion: operator.serverless.openshift.io/v1alpha1
	// kind: KnativeKafka
	// metadata:
	//   annotations:
	//     baz: fox
	//   creationTimestamp: "2009-02-13T23:31:30Z"
	//   generation: 456
	//   labels:
	//     foo: bar
	//   name: foo
	//   namespace: bar
	//   resourceVersion: "123"
	//   uid: deadbeef
	// spec:
	//   broker:
	//     defaultConfig:
	//       authSecretName: secret-name
	//       bootstrapServers: example.com
	//       numPartitions: 250
	//       replicationFactor: 150
	//     enabled: true
	//   channel:
	//     authSecretName: secret-name
	//     authSecretNamespace: secret-ns
	//     bootstrapServers: example.com
	//     enabled: false
	//   config:
	//     namespaced-broker-resources:
	//       resources: |
	//         - apiVersion: monitoring.coreos.com/v1
	//           kind: ServiceMonitor
	//           metadata:
	//             creationTimestamp: null
	//             labels:
	//               app: kafka-broker-receiver
	//             name: kafka-broker-receiver-sm
	//             namespace: '{{.Namespace}}'
	//           spec:
	//             endpoints:
	//             - bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
	//               bearerTokenSecret:
	//                 key: ""
	//               port: https
	//               scheme: https
	//               tlsConfig:
	//                 ca: {}
	//                 caFile: /etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt
	//                 cert: {}
	//                 serverName: kafka-broker-receiver-sm-service.{{.Namespace}}.svc
	//             namespaceSelector:
	//               matchNames:
	//               - '{{.Namespace}}'
	//             selector:
	//               matchLabels:
	//                 name: kafka-broker-receiver-sm-service
	//         - apiVersion: monitoring.coreos.com/v1
	//           kind: ServiceMonitor
	//           metadata:
	//             creationTimestamp: null
	//             labels:
	//               app: kafka-broker-dispatcher
	//             name: kafka-broker-dispatcher-sm
	//             namespace: '{{.Namespace}}'
	//           spec:
	//             endpoints:
	//             - bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
	//               bearerTokenSecret:
	//                 key: ""
	//               port: https
	//               scheme: https
	//               tlsConfig:
	//                 ca: {}
	//                 caFile: /etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt
	//                 cert: {}
	//                 serverName: kafka-broker-dispatcher-sm-service.{{.Namespace}}.svc
	//             namespaceSelector:
	//               matchNames:
	//               - '{{.Namespace}}'
	//             selector:
	//               matchLabels:
	//                 name: kafka-broker-dispatcher-sm-service
	//         - apiVersion: v1
	//           kind: Service
	//           metadata:
	//             annotations:
	//               service.beta.openshift.io/serving-cert-secret-name: kafka-broker-receiver-sm-service-tls
	//             creationTimestamp: null
	//             labels:
	//               name: kafka-broker-receiver-sm-service
	//             name: kafka-broker-receiver-sm-service
	//             namespace: '{{.Namespace}}'
	//           spec:
	//             ports:
	//             - name: https
	//               port: 8444
	//               targetPort: 8444
	//             selector:
	//               app: kafka-broker-receiver
	//           status:
	//             loadBalancer: {}
	//         - apiVersion: v1
	//           kind: Service
	//           metadata:
	//             annotations:
	//               service.beta.openshift.io/serving-cert-secret-name: kafka-broker-dispatcher-sm-service-tls
	//             creationTimestamp: null
	//             labels:
	//               name: kafka-broker-dispatcher-sm-service
	//             name: kafka-broker-dispatcher-sm-service
	//             namespace: '{{.Namespace}}'
	//           spec:
	//             ports:
	//             - name: https
	//               port: 8444
	//               targetPort: 8444
	//             selector:
	//               app: kafka-broker-dispatcher
	//           status:
	//             loadBalancer: {}
	//         - apiVersion: rbac.authorization.k8s.io/v1
	//           kind: ClusterRoleBinding
	//           metadata:
	//             creationTimestamp: null
	//             name: rbac-proxy-reviews-prom-rb-knative-kafka-broker-data-plane-{{.Namespace}}
	//           roleRef:
	//             apiGroup: rbac.authorization.k8s.io
	//             kind: ClusterRole
	//             name: rbac-proxy-reviews-prom
	//           subjects:
	//           - kind: ServiceAccount
	//             name: knative-kafka-broker-data-plane
	//             namespace: '{{.Namespace}}'
	//         - apiVersion: v1
	//           kind: Namespace
	//           metadata:
	//             creationTimestamp: null
	//             labels:
	//               openshift.io/cluster-monitoring: "true"
	//             name: '{{.Namespace}}'
	//           spec: {}
	//           status: {}
	//     other-config:
	//       resources: to be preserved
	//   high-availability:
	//     replicas: 25
	//   logging:
	//     level: DEBUG
	//   sink:
	//     enabled: false
	//   source:
	//     enabled: false
	//   workloads:
	//   - name: WorkloadOverride
	// status:
	//   observedGeneration: 1234
	//   version: "5678"
}
