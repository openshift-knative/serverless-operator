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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/logging"
)

const (
	EnableMonitoringEnvVar        = "ENABLE_SERVING_MONITORING_BY_DEFAULT"
	EnableMonitoringLabel         = "openshift.io/cluster-monitoring"
	ObservabilityCMName           = "observability"
	ObservabilityBackendKey       = "metrics.backend-destination"
	OpenshiftMonitoringNamespace  = "openshift-monitoring"
	prometheusRoleName            = "knative-serving-prometheus-k8s"
	prometheusClusterRoleName     = "rbac-proxy-metrics-prom"
	servingSMRbacManifestPath     = "SERVING_SM_RBAC_MANIFEST_PATH"
	servingSMResourceManifestPath = "SERVING_SM_RESOURCE_MANIFEST_PATH"
)

var (
	servingComponents = []string{"activator", "autoscaler", "autoscaler-hpa", "controller", "domain-mapping", "domainmapping-webhook", "webhook"}
)

func init() {
	builder := runtime.NewSchemeBuilder(monitoringv1.AddToScheme)
	_ = builder.AddToScheme(scheme.Scheme)
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

func LoadServingMonitoringPlatformManifests() ([]mf.Manifest, error) {
	rbacPath := os.Getenv(servingSMRbacManifestPath)
	if rbacPath == "" {
		return nil, fmt.Errorf("failed to get the Serving sm rbac manifest path")
	}
	rbacManifest, err := mf.NewManifest(rbacPath)
	if err != nil {
		return nil, err
	}
	resourcePath := os.Getenv(servingSMResourceManifestPath)
	if resourcePath == "" {
		return nil, fmt.Errorf("failed to get the Serving sm resource manifest path")
	}
	smBlueprint, err := mf.NewManifest(resourcePath)
	if err != nil {
		return nil, err
	}
	for _, c := range servingComponents {
		smManifest, err := constructServiceMonitorResourceManifests(c, smBlueprint.Resources())
		if err != nil {
			return nil, err
		}
		if smManifest != nil {
			rbacManifest = rbacManifest.Append(*smManifest)
		}
	}
	return []mf.Manifest{rbacManifest}, nil
}

func constructServiceMonitorResourceManifests(component string, resources []unstructured.Unstructured) (*mf.Manifest, error) {
	smManifest, err := mf.ManifestFrom(mf.Slice(resources))
	if err != nil {
		return nil, err
	}
	transforms := []mf.Transformer{transformSMResources(component)}
	if smManifest, err = smManifest.Transform(transforms...); err != nil {
		return nil, fmt.Errorf("unable to transform service monitor resource manifest: %w", err)
	}
	return &smManifest, nil
}

func transformSMResources(component string) mf.Transformer {
	prefix := "component"
	certAnnotation := "service.beta.openshift.io/serving-cert-secret-name"
	return func(u *unstructured.Unstructured) error {
		kind := strings.ToLower(u.GetKind())
		switch kind {
		case "servicemonitor":
			var sm = &monitoringv1.ServiceMonitor{}
			if err := scheme.Scheme.Convert(u, sm, nil); err != nil {
				return err
			}
			sm.Name = strings.Replace(sm.Name, prefix, component, 1)
			sm.Spec.Endpoints[0].TLSConfig.ServerName = strings.Replace(sm.Spec.Endpoints[0].TLSConfig.ServerName, prefix, component, 1)
			sm.Spec.Selector.MatchLabels["name"] = strings.Replace(sm.Spec.Selector.MatchLabels["name"], prefix, component, 1)
			return scheme.Scheme.Convert(sm, u, nil)
		case "service":
			var sv = &corev1.Service{}
			if err := scheme.Scheme.Convert(u, sv, nil); err != nil {
				return err
			}
			sv.Name = strings.Replace(sv.Name, prefix, component, 1)
			sv.Labels["name"] = strings.Replace(sv.Labels["name"], prefix, component, 1)
			sv.Annotations[certAnnotation] = strings.Replace(sv.Annotations[certAnnotation], prefix, component, 1)
			sv.Spec.Selector["app"] = strings.Replace(sv.Spec.Selector["app"], prefix, component, 1)
			return scheme.Scheme.Convert(sv, u, nil)
		}
		return nil
	}
}
