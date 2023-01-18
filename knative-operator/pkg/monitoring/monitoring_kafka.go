package monitoring

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	commonutil "github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

// AdditionalResourcesForNamespacedBroker creates the manifest of additional resources for the namespaced broker.
// That content is later consumed by the upstream Knative Kafka controller. It applies all the resources
// listed in that configmap in broker's namespace, whenever a new namespaced broker is created.
func AdditionalResourcesForNamespacedBroker() (string, error) {

	// For each namespaced broker dataplane, we do these:
	// - Create a Kubernetes `Service` that makes the dataplane pods accessible by Prometheus.
	//   That service basically integrates with the RBAC proxy sidecar in the dataplane pods
	//
	// - Create a `ServiceMonitor` that makes Prometheus scrape our dataplane pods.
	//   It also sets up certificates for the RBAC proxy mentioned above.
	//
	// - Create a `ClusterRoleBinding` of the `rbac-proxy-reviews-prom` `ClusterRole` for the
	//  `knative-kafka-broker-data-plane` `ServiceAccount` in the broker namespace. Otherwise, RBAC proxy cannot
	//   authorize itself.
	//
	// - Set `"openshift.io/cluster-monitoring": "true",` label on the broker namespace, so that Prometheus monitors
	//    the namespace and creates certs/secrets and also scrapes the pods
	//
	// While it can be outdated, here's a Gist that creates these resources manually:
	// https://gist.github.com/aliok/1a89600db9fcec0416302148fadba5ad

	additionalResources, err := createUnstructuredList(
		serviceMonitor("receiver"),
		serviceMonitor("dispatcher"),
		service("receiver"),
		service("dispatcher"),
		clusterRoleBinding(),
		namespace(),
	)
	if err != nil {
		return "", err
	}

	return commonutil.MarshalUnstructured(additionalResources)
}

func createUnstructuredList(objs ...runtime.Object) ([]unstructured.Unstructured, error) {
	list := make([]unstructured.Unstructured, 0)
	for _, obj := range objs {
		unstr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return []unstructured.Unstructured{}, err
		}
		list = append(list, unstructured.Unstructured{Object: unstr})
	}
	return list, nil
}

func namespace() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "{{.Namespace}}",
			Labels: map[string]string{
				"openshift.io/cluster-monitoring": "true",
			},
		},
	}
}

func clusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "rbac-proxy-reviews-prom-rb-knative-kafka-broker-data-plane-{{.Namespace}}",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "rbac-proxy-reviews-prom",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "knative-kafka-broker-data-plane",
			Namespace: "{{.Namespace}}",
		}},
	}
}

func serviceMonitor(component string) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "{{.Namespace}}",
			Name:      fmt.Sprintf("kafka-broker-%s-sm", component),
			Labels: map[string]string{
				"app": fmt.Sprintf("kafka-broker-%s", component),
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					BearerTokenSecret: corev1.SecretKeySelector{
						Key: "",
					},
					Port:   "https",
					Scheme: "https",
					TLSConfig: &monitoringv1.TLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							CA:         monitoringv1.SecretOrConfigMap{},
							Cert:       monitoringv1.SecretOrConfigMap{},
							ServerName: fmt.Sprintf("kafka-broker-%s-sm-service.{{.Namespace}}.svc", component),
						},
						CAFile: "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
					},
				},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{"{{.Namespace}}"},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"name": fmt.Sprintf("kafka-broker-%s-sm-service", component)},
			},
		},
	}
}

func service(component string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "{{.Namespace}}",
			Name:      fmt.Sprintf("kafka-broker-%s-sm-service", component),
			Labels: map[string]string{
				"name": fmt.Sprintf("kafka-broker-%s-sm-service", component),
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": fmt.Sprintf("kafka-broker-%s-sm-service-tls", component),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "https",
				Port:       8444,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8444},
			}},
			Selector: map[string]string{
				"app": fmt.Sprintf("kafka-broker-%s", component),
			},
		},
	}
}
