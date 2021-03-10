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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/logging"
)

const (
	EnableMonitoringEnvVar       = "ENABLE_SERVING_MONITORING_BY_DEFAULT"
	EnableMonitoringLabel        = "openshift.io/cluster-monitoring"
	ObservabilityCMName          = "observability"
	ObservabilityBackendKey      = "metrics.backend-destination"
	OpenshiftMonitoringNamespace = "openshift-monitoring"
	prometheusRoleName           = "knative-serving-prometheus-k8s"
	prometheusClusterRoleName    = "rbac-proxy-metrics-prom"
	servingSMRbacManifestPath    = "SERVING_SERVICE_MONITOR_RBAC_MANIFEST_PATH"
)

var (
	servingComponents = sets.NewString("activator", "autoscaler", "autoscaler-hpa", "controller", "domain-mapping", "domainmapping-webhook", "webhook")
)

func init() {
	_ = monitoringv1.AddToScheme(scheme.Scheme)
}

func ReconcileServingMonitoring(ctx context.Context, api kubernetes.Interface, ks *v1alpha1.KnativeServing) error {
	backend := ks.Spec.CommonSpec.Config[ObservabilityCMName][ObservabilityBackendKey]
	if shouldEnableMonitoring(backend) {
		if err := reconcileMonitoringLabelOnNamespace(ctx, ks.Namespace, api, true); err != nil {
			return fmt.Errorf("failed to enable monitoring %w ", err)
		}
		return nil
	}
	if err := reconcileMonitoringLabelOnNamespace(ctx, ks.Namespace, api, false); err != nil {
		return fmt.Errorf("failed to disable monitoring %w ", err)
	}
	common.Configure(&ks.Spec.CommonSpec, ObservabilityCMName, ObservabilityBackendKey, "none")
	return nil
}

func shouldEnableMonitoring(backend string) bool {
	if backend == "none" {
		return false
	}
	enable, present := os.LookupEnv(EnableMonitoringEnvVar)
	// Skip setup from env if feature toggle is not present, use whatever the user defines in the Serving CR.
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
		log.Info("Enabling Serving monitoring")
	} else {
		log.Info("Disabling Serving monitoring")
	}
	if ns.Labels == nil {
		ns.Labels = make(map[string]string, 1)
	}
	ns.Labels[EnableMonitoringLabel] = strconv.FormatBool(enable)
	if _, err := api.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{}); err != nil {
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
		} else if (kind == "clusterrolebinding" && u.GetName() == prometheusClusterRoleName) || (kind == "rolebinding" && u.GetName() == prometheusRoleName) {
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

func LoadServingMonitoringPlatformManifests(ns string) ([]mf.Manifest, error) {
	rbacPath := os.Getenv(servingSMRbacManifestPath)
	if rbacPath == "" {
		return nil, fmt.Errorf("failed to get the Serving sm rbac manifest path")
	}
	rbacManifest, err := mf.NewManifest(rbacPath)
	if err != nil {
		return nil, err
	}
	for c := range servingComponents {
		smManifest, err := constructServiceMonitorResourceManifests(c, ns)
		if err != nil {
			return nil, err
		}
		if smManifest != nil {
			rbacManifest = rbacManifest.Append(*smManifest)
		}
	}
	return []mf.Manifest{rbacManifest}, nil
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
						ServerName: fmt.Sprintf("%s.knative-serving.svc", serviceName),
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
			Selector: map[string]string{"app": component},
		}}
}
