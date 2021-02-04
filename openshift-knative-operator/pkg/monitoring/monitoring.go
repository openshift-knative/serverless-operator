package monitoring

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const (
	DisableMonitoringEnvVar      = "DISABLE_SERVING_MONITORING_BY_DEFAULT"
	EnableMonitoringLabel        = "openshift.io/cluster-monitoring"
	ObservabilityCMName          = "observability"
	ObservabilityBackendKey      = "metrics.backend-destination"
	OpenshiftMonitoringNamespace = "openshift-monitoring"
)

func SetupServingMonitoring(api kubernetes.Interface, ks *v1alpha1.KnativeServing, log *zap.SugaredLogger) error {
	if shouldDisableMonitoring(ks.Spec.CommonSpec.Config) {
		log.Info("Disabling Serving monitoring")
		if err := disableMonitoring(api, ks); err != nil {
			return fmt.Errorf("failed to disable monitoring %w ", err)
		}
		return nil
	}
	if err := setMonitoringLabelToNamespace(ks.Namespace, api, true); err != nil {
		return err
	}
	return nil
}

func shouldDisableMonitoring(cfg v1alpha1.ConfigMapData) bool {
	_, disableByDefault := os.LookupEnv(DisableMonitoringEnvVar)
	backend, backendIsSet := cfg[ObservabilityCMName][ObservabilityBackendKey]
	return (disableByDefault && !backendIsSet) || backend == "none"
}

func disableMonitoring(api kubernetes.Interface, ks *v1alpha1.KnativeServing) error {
	// Disable metrics backend by default in case we need to eg. SRVKS-679
	if _, ok := ks.Spec.CommonSpec.Config[ObservabilityCMName][ObservabilityBackendKey]; !ok {
		common.Configure(&ks.Spec.CommonSpec, ObservabilityCMName, ObservabilityBackendKey, "none")
	}
	return setMonitoringLabelToNamespace(ks.Namespace, api, false)
}

func setMonitoringLabelToNamespace(namespace string, api kubernetes.Interface, enable bool) error {
	ns, err := api.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	ns.Labels[EnableMonitoringLabel] = strconv.FormatBool(enable)
	if _, err := api.CoreV1().Namespaces().Update(context.Background(), ns, metav1.UpdateOptions{}); err != nil {
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
		if kind == "role" && u.GetName() == "knative-serving-prometheus-k8s" {
			u.SetNamespace(resourceNamespace)
		} else if (kind == "clusterrolebinding" && u.GetName() == "rbac-proxy-metrics-prom") || (kind == "rolebinding" && u.GetName() == "knative-serving-prometheus-k8s") {
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
