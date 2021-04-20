package monitoring

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

var (
	servingDeployments = sets.NewString("activator", "autoscaler", "autoscaler-hpa", "controller", "domain-mapping", "domainmapping-webhook", "webhook")
)

func ReconcileMonitoringForServing(ctx context.Context, api kubernetes.Interface, ks *v1alpha1.KnativeServing) error {
	return reconcileMonitoring(ctx, api, &ks.Spec.CommonSpec, ks.GetNamespace())
}

func GetServingTransformers(comp v1alpha1.KComponent) []mf.Transformer {
	if shouldEnableMonitoring(comp.GetSpec().GetConfig()) {
		return []mf.Transformer{
			injectNamespaceWithSubject(comp.GetNamespace(), OpenshiftMonitoringNamespace),
			injectRbacProxyContainerToDeployments(servingDeployments),
		}
	}
	return []mf.Transformer{}
}

func GetServingMonitoringPlatformManifests(ks v1alpha1.KComponent) ([]mf.Manifest, error) {
	rbacManifest, err := getRBACManifest()
	if err != nil {
		return nil, err
	}
	// Serving has one common sa for all pods
	crbM, err := createClusterRoleBindingManifest("controller", ks.GetNamespace())
	if err != nil {
		return nil, err
	}
	rbacManifest = rbacManifest.Append(*crbM)
	for c := range servingDeployments {
		if err := appendManifestsForComponent(c, ks.GetNamespace(), &rbacManifest); err != nil {
			return nil, err
		}
	}
	return []mf.Manifest{rbacManifest}, nil
}
