package monitoring

import (
	"fmt"
)

func ExampleAdditionalResourcesForNamespacedBroker() {
	str, err := AdditionalResourcesForNamespacedBroker()
	if err != nil {
		fmt.Printf("AdditionalResourcesForNamespacedBroker() error: %v", err)
	}

	fmt.Println(str)
	// Output:
	// - apiVersion: monitoring.coreos.com/v1
	//   kind: ServiceMonitor
	//   metadata:
	//     creationTimestamp: null
	//     labels:
	//       app: kafka-broker-receiver
	//     name: kafka-broker-receiver-sm
	//     namespace: '{{.Namespace}}'
	//   spec:
	//     endpoints:
	//     - bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
	//       bearerTokenSecret:
	//         key: ""
	//       port: https
	//       scheme: https
	//       tlsConfig:
	//         ca: {}
	//         caFile: /etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt
	//         cert: {}
	//         serverName: kafka-broker-receiver-sm-service.{{.Namespace}}.svc
	//     namespaceSelector:
	//       matchNames:
	//       - '{{.Namespace}}'
	//     selector:
	//       matchLabels:
	//         name: kafka-broker-receiver-sm-service
	// - apiVersion: monitoring.coreos.com/v1
	//   kind: ServiceMonitor
	//   metadata:
	//     creationTimestamp: null
	//     labels:
	//       app: kafka-broker-dispatcher
	//     name: kafka-broker-dispatcher-sm
	//     namespace: '{{.Namespace}}'
	//   spec:
	//     endpoints:
	//     - bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
	//       bearerTokenSecret:
	//         key: ""
	//       port: https
	//       scheme: https
	//       tlsConfig:
	//         ca: {}
	//         caFile: /etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt
	//         cert: {}
	//         serverName: kafka-broker-dispatcher-sm-service.{{.Namespace}}.svc
	//     namespaceSelector:
	//       matchNames:
	//       - '{{.Namespace}}'
	//     selector:
	//       matchLabels:
	//         name: kafka-broker-dispatcher-sm-service
	// - apiVersion: v1
	//   kind: Service
	//   metadata:
	//     annotations:
	//       service.beta.openshift.io/serving-cert-secret-name: kafka-broker-receiver-sm-service-tls
	//     creationTimestamp: null
	//     labels:
	//       name: kafka-broker-receiver-sm-service
	//     name: kafka-broker-receiver-sm-service
	//     namespace: '{{.Namespace}}'
	//   spec:
	//     ports:
	//     - name: https
	//       port: 8444
	//       targetPort: 8444
	//     selector:
	//       app: kafka-broker-receiver
	//   status:
	//     loadBalancer: {}
	// - apiVersion: v1
	//   kind: Service
	//   metadata:
	//     annotations:
	//       service.beta.openshift.io/serving-cert-secret-name: kafka-broker-dispatcher-sm-service-tls
	//     creationTimestamp: null
	//     labels:
	//       name: kafka-broker-dispatcher-sm-service
	//     name: kafka-broker-dispatcher-sm-service
	//     namespace: '{{.Namespace}}'
	//   spec:
	//     ports:
	//     - name: https
	//       port: 8444
	//       targetPort: 8444
	//     selector:
	//       app: kafka-broker-dispatcher
	//   status:
	//     loadBalancer: {}
	// - apiVersion: rbac.authorization.k8s.io/v1
	//   kind: ClusterRoleBinding
	//   metadata:
	//     creationTimestamp: null
	//     name: rbac-proxy-reviews-prom-rb-knative-kafka-broker-data-plane-{{.Namespace}}
	//   roleRef:
	//     apiGroup: rbac.authorization.k8s.io
	//     kind: ClusterRole
	//     name: rbac-proxy-reviews-prom
	//   subjects:
	//   - kind: ServiceAccount
	//     name: knative-kafka-broker-data-plane
	//     namespace: '{{.Namespace}}'
	// - apiVersion: v1
	//   kind: Namespace
	//   metadata:
	//     creationTimestamp: null
	//     labels:
	//       openshift.io/cluster-monitoring: "true"
	//     name: '{{.Namespace}}'
	//   spec: {}
	//   status: {}
}
