package monitoring

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/logging"
)

const (
	EnableServingMonitoringEnvVar  = "ENABLE_SERVING_MONITORING_BY_DEFAULT"
	EnableEventingMonitoringEnvVar = "ENABLE_EVENTING_MONITORING_BY_DEFAULT"
	EnableMonitoringLabel          = "openshift.io/cluster-monitoring"
	ObservabilityCMName            = "observability"
	ObservabilityBackendKey        = "metrics.backend-destination"
	OpenshiftMonitoringNamespace   = "openshift-monitoring"
	prometheusRoleName             = "knative-prometheus-k8s"
	prometheusClusterRoleName      = "rbac-proxy-metrics-prom"
	smRbacManifestPath             = "SERVICE_MONITOR_RBAC_MANIFEST_PATH"
)

type KComponentType string

const (
	Serving  KComponentType = "Serving"
	Eventing KComponentType = "Eventing"
)

var (
	servingComponents  = sets.NewString("activator", "autoscaler", "autoscaler-hpa", "controller", "domain-mapping", "domainmapping-webhook", "webhook")
	eventingComponents = sets.NewString("eventing-controller", "eventing-webhook", "imc-controller", "imc-dispatcher", "mt-broker-controller", "mt-broker-filter", "mt-broker-ingress", "sugar-controller")
)

func init() {
	_ = monitoringv1.AddToScheme(scheme.Scheme)
}

func ReconcileMonitoringForKComponent(ctx context.Context, api kubernetes.Interface, comp v1alpha1.KComponent) error {
	commonSpec, cType := getCommonSpec(comp)
	if shouldEnableMonitoring(commonSpec, cType) {
		if err := reconcileMonitoringLabelOnNamespace(ctx, comp.GetNamespace(), api, true, cType); err != nil {
			return fmt.Errorf("failed to enable monitoring %w ", err)
		}
		return nil
	}
	// If "opencensus" is used we still dont want to scrape from a Serverless controlled namespace
	// user can always push to an agent collector in some other namespace and then integrate with OCP monitoring stack
	if err := reconcileMonitoringLabelOnNamespace(ctx, comp.GetNamespace(), api, false, cType); err != nil {
		return fmt.Errorf("failed to disable monitoring %w ", err)
	}
	common.Configure(commonSpec, ObservabilityCMName, ObservabilityBackendKey, "none")
	return nil
}

func getCommonSpec(comp v1alpha1.KComponent) (*v1alpha1.CommonSpec, KComponentType) {
	switch c := comp.(type) {
	case *v1alpha1.KnativeServing:
		return &c.Spec.CommonSpec, Serving
	case *v1alpha1.KnativeEventing:
		return &c.Spec.CommonSpec, Eventing
	}
	return nil, ""
}

func shouldEnableMonitoring(commonSpec *v1alpha1.CommonSpec, comp KComponentType) bool {
	backend := commonSpec.Config[ObservabilityCMName][ObservabilityBackendKey]
	if backend == "none" || backend == "opencensus" {
		return false
	}
	var enable string
	var present bool

	switch comp {
	case Serving:
		enable, present = os.LookupEnv(EnableServingMonitoringEnvVar)
	case Eventing:
		enable, present = os.LookupEnv(EnableEventingMonitoringEnvVar)
	}

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

func reconcileMonitoringLabelOnNamespace(ctx context.Context, namespace string, api kubernetes.Interface, enable bool, cType KComponentType) error {
	ns, err := api.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if ns.Labels[EnableMonitoringLabel] == strconv.FormatBool(enable) {
		return nil
	}
	log := logging.FromContext(ctx)
	if enable {
		log.Infof("Enabling %s monitoring", cType)
	} else {
		log.Infof("Disabling %s monitoring", cType)
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

// InjectNamespaceWithSubject uses a custom transformation to avoid operator overriding everything with the current namespace including
// subjects ns. Here we break the assumption of the operator about all resources being in the same namespace
// since we need to setup RBAC for the prometheus-k8s account which resides in openshift-monitoring ns.
func InjectNamespaceWithSubject(resourceNamespace string, subjectNamespace string) mf.Transformer {
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

func getMonitoringPlarformNanifests(ns string, cType KComponentType) ([]mf.Manifest, error) {
	rbacPath := os.Getenv(smRbacManifestPath)
	if rbacPath == "" {
		return nil, fmt.Errorf("failed to get the Serving sm rbac manifest path")
	}
	rbacManifest, err := mf.NewManifest(rbacPath)
	if err != nil {
		return nil, err
	}
	switch cType {
	case Serving:
		// Serving has one common sa for all pods
		crbM, err := createClusterRoleBindingManifest("controller", ns)
		if err != nil {
			return nil, err
		}
		rbacManifest = rbacManifest.Append(*crbM)
		for c := range servingComponents {
			if err := appendManifestsForComponent(c, ns, &rbacManifest); err != nil {
				return nil, err
			}
		}
	case Eventing:
		// Only mt-broker-controller has a different than its name sa (eventing-controller)
		for sa := range eventingComponents {
			if sa == "mt-broker-controller" {
				continue
			}
			crbM, err := createClusterRoleBindingManifest(sa, ns)
			if err != nil {
				return nil, err
			}
			rbacManifest = rbacManifest.Append(*crbM)
		}
		for c := range eventingComponents {
			if err := appendManifestsForComponent(c, ns, &rbacManifest); err != nil {
				return nil, err
			}
		}
	}

	return []mf.Manifest{rbacManifest}, nil
}

func appendManifestsForComponent(c string, ns string, rbacManifest *mf.Manifest) error {
	smManifest, err := constructServiceMonitorResourceManifests(c, ns)
	if err != nil {
		return err
	}
	if smManifest != nil {
		*rbacManifest = rbacManifest.Append(*smManifest)
	}
	return nil
}

func GetCompMonitoringPlatformManifests(comp v1alpha1.KComponent) ([]mf.Manifest, error) {
	cSpec, cType := getCommonSpec(comp)
	if shouldEnableMonitoring(cSpec, cType) {
		return getMonitoringPlarformNanifests(comp.GetNamespace(), cType)
	}
	return []mf.Manifest{}, nil
}

func GetCompTransformers(comp v1alpha1.KComponent) []mf.Transformer {
	cSpec, cType := getCommonSpec(comp)
	if shouldEnableMonitoring(cSpec, cType) {
		return []mf.Transformer{
			InjectNamespaceWithSubject(comp.GetNamespace(), OpenshiftMonitoringNamespace),
			InjectRbacProxyContainerToDeployments(),
		}
	}
	return []mf.Transformer{}
}

func constructServiceMonitorResourceManifests(component string, ns string) (*mf.Manifest, error) {
	var smU = &unstructured.Unstructured{}
	var svU = &unstructured.Unstructured{}
	sms := createServiceMonitorService(component)
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
			Name: fmt.Sprintf("%s-sm", component),
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

func createServiceMonitorService(component string) corev1.Service {
	serviceName := fmt.Sprintf("%s-sm-service", component)
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        serviceName,
			Labels:      map[string]string{"name": serviceName},
			Annotations: map[string]string{"service.beta.openshift.io/serving-cert-secret-name": fmt.Sprintf("%s-tls", serviceName)},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name: "https",
				Port: 8444,
			}},
			Selector: getSelectorLabels(component),
		}}
}

func createClusterRoleBindingManifest(serviceAccountName string, ns string) (*mf.Manifest, error) {
	crb := v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("rbac-proxy-reviews-prom-rb-%s", serviceAccountName),
		},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "rbac-proxy-reviews-prom",
		},
		Subjects: []v1.Subject{{
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
	if servingComponents.Has(name) {
		return "9090"
	}
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
	case "sugar-controller":
		labels["eventing.knative.dev/role"] = component
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
	default:
		labels["app"] = component
	}
	return labels
}
