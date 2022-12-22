package monitoring

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	serverlessoperatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	commonutil "github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

func ReconcileMonitoringForNamespacedBroker(kk *serverlessoperatorv1alpha1.KnativeKafka) error {
	additionalResources, err := createUnstructuredList(
		receiverServiceMonitor(),
		dispatcherServiceMonitor(),
		receiverService(),
		dispatcherService(),
		clusterRoleBinding(),
		namespace(),
	)
	if err != nil {
		return err
	}

	resStr, err := commonutil.MarshalUnstructured(additionalResources)
	if err != nil {
		return err
	}

	if len(kk.Spec.Config) == 0 {
		kk.Spec.Config = make(map[string]map[string]string, 1)
	}

	// TODO: create constants
	if len(kk.Spec.Config["namespaced-broker-resources"]) == 0 {
		kk.Spec.Config["namespaced-broker-resources"] = make(map[string]string, 1)
	}

	kk.Spec.Config["namespaced-broker-resources"]["resources"] = resStr
	return nil
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

// TODO: create constants
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

func receiverServiceMonitor() *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "{{.Namespace}}",
			Name:      "kafka-broker-receiver-sm",
			Labels: map[string]string{
				"app": "kafka-broker-receiver",
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
							ServerName: "kafka-broker-receiver-sm-service.{{.Namespace}}.svc",
						},
						CAFile: "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
					},
				},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{"{{.Namespace}}"},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"name": "kafka-broker-receiver-sm-service"},
			},
		},
	}
}

func dispatcherServiceMonitor() *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "{{.Namespace}}",
			Name:      "kafka-broker-dispatcher-sm",
			Labels: map[string]string{
				"app": "kafka-broker-dispatcher",
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
							ServerName: "kafka-broker-dispatcher-sm-service.{{.Namespace}}.svc",
						},
						CAFile: "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
					},
				},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{"{{.Namespace}}"},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"name": "kafka-broker-dispatcher-sm-service"},
			},
		},
	}
}

func receiverService() *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "{{.Namespace}}",
			Name:      "kafka-broker-receiver-sm-service",
			Labels: map[string]string{
				"name": "kafka-broker-receiver-sm-service",
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": "kafka-broker-receiver-sm-service-tls",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "https",
				Port:       8444,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8444},
			}},
			Selector: map[string]string{
				"app": "kafka-broker-receiver",
			},
		},
	}
}

func dispatcherService() *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "{{.Namespace}}",
			Name:      "kafka-broker-dispatcher-sm-service",
			Labels: map[string]string{
				"name": "kafka-broker-dispatcher-sm-service",
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": "kafka-broker-dispatcher-sm-service-tls",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "https",
				Port:       8444,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8444},
			}},
			Selector: map[string]string{
				"app": "kafka-broker-dispatcher",
			},
		},
	}
}
