package monitoring

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	mf "github.com/manifestival/manifestival"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/logging"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
)

const (
	EnableMonitoringEnvVar       = "ENABLE_MONITORING_BY_DEFAULT"
	EnableMonitoringLabel        = "openshift.io/cluster-monitoring"
	ObservabilityCMName          = "observability"
	ObservabilityBackendKey      = "metrics.backend-destination"
	OpenshiftMonitoringNamespace = "openshift-monitoring"
	prometheusRoleName           = "knative-prometheus-k8s"
	prometheusClusterRoleName    = "rbac-proxy-metrics-prom"
	smRbacManifestPath           = "SERVICE_MONITOR_RBAC_MANIFEST_PATH"
)

func init() {
	_ = monitoringv1.AddToScheme(scheme.Scheme)
}

// injectNamespaceWithSubject uses a custom transformation to avoid operator overriding everything with the current namespace including
// subjects ns. Here we break the assumption of the operator about all resources being in the same namespace
// since we need to setup RBAC for the prometheus-k8s account which resides in openshift-monitoring ns.
func injectNamespaceWithSubject(resourceNamespace string, subjectNamespace string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind := strings.ToLower(u.GetKind())
		// Only touch the related manifests.
		if kind == "role" && u.GetName() == prometheusRoleName {
			u.SetNamespace(resourceNamespace)
		} else if (kind == "clusterrolebinding" && u.GetName() == prometheusClusterRoleName+"-rb") || (kind == "rolebinding" && u.GetName() == prometheusRoleName) {
			if kind == "rolebinding" {
				u.SetNamespace(resourceNamespace)
			}
			subjects, _, _ := unstructured.NestedFieldNoCopy(u.Object, "subjects")
			for _, subject := range subjects.([]interface{}) {
				m := subject.(map[string]interface{})
				if _, ok := m["namespace"]; ok {
					m["namespace"] = subjectNamespace
				}
			}
		}
		return nil
	}
}

func reconcileMonitoring(ctx context.Context, api kubernetes.Interface, spec *operatorv1alpha1.CommonSpec, ns string) error {
	if ShouldEnableMonitoring(spec.GetConfig()) {
		if err := reconcileMonitoringLabelOnNamespace(ctx, ns, api, true); err != nil {
			return fmt.Errorf("failed to enable monitoring %w ", err)
		}
		return nil
	}
	// If "opencensus" is used we still dont want to scrape from a Serverless controlled namespace
	// user can always push to an agent collector in some other namespace and then integrate with OCP monitoring stack
	if err := reconcileMonitoringLabelOnNamespace(ctx, ns, api, false); err != nil {
		return fmt.Errorf("failed to disable monitoring %w ", err)
	}
	common.Configure(spec, ObservabilityCMName, ObservabilityBackendKey, "none")
	return nil
}

func ShouldEnableMonitoring(config operatorv1alpha1.ConfigMapData) bool {
	backend := config[ObservabilityCMName][ObservabilityBackendKey]
	if backend == "none" || backend == "opencensus" {
		return false
	}

	var enable string
	var present bool

	enable, present = os.LookupEnv(EnableMonitoringEnvVar)
	// Skip setup from env if feature toggle is not present, use whatever the user defines in the comp CR.
	if !present {
		return true
	}
	parsedEnable := strings.EqualFold(enable, "true")
	// Let the user enable monitoring with a proper backend value even if feature toggle is off.
	if !parsedEnable && backend != "" {
		return true
	}
	return parsedEnable
}

func reconcileMonitoringLabelOnNamespace(ctx context.Context, namespace string, api kubernetes.Interface, enable bool) error {
	ns, err := api.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if ns.Labels[EnableMonitoringLabel] == strconv.FormatBool(enable) {
		return nil
	}
	log := logging.FromContext(ctx)
	if enable {
		log.Infof("Enabling monitoring")
	} else {
		log.Infof("Disabling monitoring")
	}
	if ns.Labels == nil {
		ns.Labels = make(map[string]string, 1)
	}
	ns.Labels[EnableMonitoringLabel] = strconv.FormatBool(enable)
	if _, err = api.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("could not add label %q to namespace %q: %w", EnableMonitoringLabel, namespace, err)
	}
	return nil
}

func AppendManifestsForComponent(c string, ns string, rbacManifest *mf.Manifest) error {
	smManifest, err := constructServiceMonitorResourceManifests(c, ns)
	if err != nil {
		return err
	}
	if smManifest != nil {
		*rbacManifest = rbacManifest.Append(*smManifest)
	}
	return nil
}

func constructServiceMonitorResourceManifests(component string, ns string) (*mf.Manifest, error) {
	var smU = &unstructured.Unstructured{}
	var svU = &unstructured.Unstructured{}
	sms := createServiceMonitorService(component, ns)
	if err := scheme.Scheme.Convert(&sms, svU, nil); err != nil {
		return nil, err
	}
	sm := createServiceMonitor(component, ns, sms.Name)
	if err := scheme.Scheme.Convert(&sm, smU, nil); err != nil {
		return nil, err
	}
	smManifest, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{*smU, *svU}))
	if err != nil {
		return nil, err
	}
	return &smManifest, nil
}

func createServiceMonitor(component string, ns string, serviceName string) monitoringv1.ServiceMonitor {
	return monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sm", component),
			Namespace: ns,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				BearerTokenFile:   "/var/run/secrets/kubernetes.io/serviceaccount/token",
				BearerTokenSecret: corev1.SecretKeySelector{Key: ""},
				Port:              "https",
				Scheme:            "https",
				TLSConfig: &monitoringv1.TLSConfig{
					CAFile: "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
					SafeTLSConfig: monitoringv1.SafeTLSConfig{
						ServerName: fmt.Sprintf("%s.%s.svc", serviceName, ns),
					},
				},
			},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{ns},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"name": serviceName},
			},
		}}
}

func createServiceMonitorService(component string, ns string) corev1.Service {
	serviceName := fmt.Sprintf("%s-sm-service", component)
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        serviceName,
			Namespace:   ns,
			Labels:      map[string]string{"name": serviceName},
			Annotations: map[string]string{"service.beta.openshift.io/serving-cert-secret-name": fmt.Sprintf("%s-tls", serviceName)},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "https",
				Port:       8444,
				TargetPort: intstr.FromInt(8444),
			}},
			Selector: getSelectorLabels(component),
		}}
}

func CreateClusterRoleBindingManifest(serviceAccountName string, ns string) (*mf.Manifest, error) {
	crb := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rbac-proxy-reviews-prom-rb-%s", serviceAccountName),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "rbac-proxy-reviews-prom",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccountName,
			Namespace: ns,
		}},
	}
	var crbU = &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(&crb, crbU, nil); err != nil {
		return nil, err
	}
	manifest, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{*crbU}))
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}

// getDefaultMetricsPort returns the expected metrics port under the assumption that this will not change
// This is static information since observability cm does not allow any changes for the prometheus config
// TODO(skonto): fix this upstream so ports are aligned if possible
func getDefaultMetricsPort(name string) string {
	if name == "mt-broker-ingress" || name == "mt-broker-filter" {
		return "9092"
	}
	return "9090"
}

// getSelectorLabels returns the correct deployment label to use with the service monitor service.
// The component is any Serving, Eventing component name eg. activator. Each component's deployment
// has either a label "app" or some special label or a set of labels that are unique to the component.
func getSelectorLabels(component string) map[string]string {
	labels := map[string]string{}
	switch component {
	case "imc-controller":
		labels["messaging.knative.dev/channel"] = "in-memory-channel"
		labels["messaging.knative.dev/role"] = "controller"
	case "imc-dispatcher":
		labels["messaging.knative.dev/channel"] = "in-memory-channel"
		labels["messaging.knative.dev/role"] = "dispatcher"
	case "mt-broker-filter":
		labels["eventing.knative.dev/brokerRole"] = "filter"
	case "mt-broker-ingress":
		labels["eventing.knative.dev/brokerRole"] = "ingress"
	case "kafka-controller-manager":
		labels["control-plane"] = "kafka-controller-manager"
	default:
		labels["app"] = component
	}
	return labels
}

func getRBACManifest() (mf.Manifest, error) {
	rbacPath := os.Getenv(smRbacManifestPath)
	if rbacPath == "" {
		return mf.Manifest{}, fmt.Errorf("failed to get the Serving sm rbac manifest path")
	}
	rbacManifest, err := mf.NewManifest(rbacPath)
	return rbacManifest, err
}
