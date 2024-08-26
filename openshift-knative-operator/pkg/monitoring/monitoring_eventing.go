package monitoring

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
)

var (
	eventingDeployments = sets.New[string](
		"eventing-controller",
		"eventing-istio-controller",
		"eventing-webhook",
		"imc-controller",
		"imc-dispatcher",
		"mt-broker-controller",
		"mt-broker-filter",
		"mt-broker-ingress",
		"pingsource-mt-adapter",
		"job-sink",
	)
)

func ReconcileMonitoringForEventing(ctx context.Context, api kubernetes.Interface, ke *operatorv1beta1.KnativeEventing) error {
	return reconcileMonitoring(ctx, api, &ke.Spec.CommonSpec, ke.GetNamespace())
}

func GetEventingTransformers(comp base.KComponent) []mf.Transformer {
	// When monitoring is off we keep around the required resources, only rbac-proxy is removed
	transformers := []mf.Transformer{injectNamespaceWithSubject(comp.GetNamespace(), OpenshiftMonitoringNamespace)}
	if ShouldEnableMonitoring(comp.GetSpec().GetConfig()) {
		transformers = append(transformers, InjectRbacProxyContainer(eventingDeployments, comp.GetSpec().GetConfig()))
		transformers = append(transformers, ExtensionDeploymentOverrides(comp.GetSpec().GetWorkloadOverrides(), eventingDeployments))
	}
	return transformers
}

func GetEventingMonitoringPlatformManifests(ke base.KComponent) ([]mf.Manifest, error) {
	rbacManifest, err := getRBACManifest()
	if err != nil {
		return nil, err
	}
	deployments := eventingDeployments
	if !isJobSinkSupported(ke) {
		deployments = eventingDeployments.Clone()
		deployments.Delete("job-sink")
	}

	// Only mt-broker-controller has a different than its name sa (eventing-controller)
	for sa := range deployments {
		if sa == "mt-broker-controller" {
			continue
		}
		crbM, err := CreateClusterRoleBindingManifest(sa, ke.GetNamespace())
		if err != nil {
			return nil, err
		}
		rbacManifest = rbacManifest.Append(*crbM)
	}
	for c := range deployments {
		if err := AppendManifestsForComponent(c, ke.GetNamespace(), &rbacManifest); err != nil {
			return nil, err
		}
	}
	return []mf.Manifest{rbacManifest}, nil
}

func isJobSinkSupported(ke base.KComponent) bool {
	if manifest, err := common.TargetManifest(ke); err == nil {
		for _, u := range manifest.Resources() {
			if u.GetName() == "job-sink" && u.GetKind() == "Deployment" {
				return true
			}
		}
	}
	return false
}
