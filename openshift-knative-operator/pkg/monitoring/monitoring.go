package monitoring

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/logging"
)

const (
	EnableMonitoringEnvVar       = "ENABLE_SERVING_MONITORING_BY_DEFAULT"
	EnableMonitoringLabel        = "openshift.io/cluster-monitoring"
	ObservabilityCMName          = "observability"
	ObservabilityBackendKey      = "metrics.backend-destination"
	OpenshiftMonitoringNamespace = "openshift-monitoring"
)

const (
	prometheusRoleName        = "knative-serving-prometheus-k8s"
	prometheusClusterRoleName = "rbac-proxy-metrics-prom"
)

func ReconcileServingMonitoring(ctx context.Context, api kubernetes.Interface, ks *v1alpha1.KnativeServing) error {
	backend, isSet := ks.Spec.CommonSpec.Config[ObservabilityCMName][ObservabilityBackendKey]
	log := logging.FromContext(ctx)
	if shouldEnableMonitoring(backend) {
		log.Info("Enabling Serving monitoring")
		if err := reconcileMonitoringLabelOnNamespace(ctx, ks.Namespace, api, true); err != nil {
			return fmt.Errorf("failed to enable monitoring %w ", err)
		}
		return nil
	}
	log.Info("Disabling Serving monitoring")
	if err := reconcileMonitoringLabelOnNamespace(ctx, ks.Namespace, api, false); err != nil {
		return err
	}
	if !isSet {
		common.Configure(&ks.Spec.CommonSpec, ObservabilityCMName, ObservabilityBackendKey, "none")
	}
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
	if ns.Labels == nil {
		ns.Labels = make(map[string]string, 1)
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
