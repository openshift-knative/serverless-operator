package monitoring

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

var (
	eventingDeployments = sets.NewString("eventing-controller", "eventing-webhook", "imc-controller", "imc-dispatcher", "mt-broker-controller", "mt-broker-filter", "mt-broker-ingress", "sugar-controller")
)

func ReconcileMonitoringForEventing(ctx context.Context, api kubernetes.Interface, ke *v1alpha1.KnativeEventing) error {
	return reconcileMonitoring(ctx, api, &ke.Spec.CommonSpec, ke.GetNamespace())
}

func GetEventingTransformers(comp v1alpha1.KComponent) []mf.Transformer {
	if shouldEnableMonitoring(comp.GetSpec().GetConfig()) {
		return []mf.Transformer{
			injectNamespaceWithSubject(comp.GetNamespace(), OpenshiftMonitoringNamespace),
			injectRbacProxyContainerToDeployments(eventingDeployments),
		}
	}
	return []mf.Transformer{}
}

func GetEventingMonitoringPlatformManifests(ke v1alpha1.KComponent) ([]mf.Manifest, error) {
	rbacManifest, err := getRBACManifest()
	if err != nil {
		return nil, err
	}
	// Only mt-broker-controller has a different than its name sa (eventing-controller)
	for sa := range eventingDeployments {
		if sa == "mt-broker-controller" {
			continue
		}
		crbM, err := createClusterRoleBindingManifest(sa, ke.GetNamespace())
		if err != nil {
			return nil, err
		}
		rbacManifest = rbacManifest.Append(*crbM)
	}
	for c := range eventingDeployments {
		if err := appendManifestsForComponent(c, ke.GetNamespace(), &rbacManifest); err != nil {
			return nil, err
		}
	}

	return []mf.Manifest{rbacManifest}, nil
}
