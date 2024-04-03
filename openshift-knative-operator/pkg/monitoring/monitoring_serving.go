package monitoring

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

var (
	servingDeployments = sets.New[string]("activator", "autoscaler", "autoscaler-hpa", "controller", "webhook")
)

func ReconcileMonitoringForServing(ctx context.Context, api kubernetes.Interface, ks *operatorv1beta1.KnativeServing) error {
	return reconcileMonitoring(ctx, api, &ks.Spec.CommonSpec, ks.GetNamespace())
}

func GetServingTransformers(comp base.KComponent) []mf.Transformer {
	// When monitoring is off we keep around the required resources, only rbac-proxy is removed
	transformers := []mf.Transformer{injectNamespaceWithSubject(comp.GetNamespace(), OpenshiftMonitoringNamespace)}
	if ShouldEnableMonitoring(comp.GetSpec().GetConfig()) {
		transformers = append(transformers, InjectRbacProxyContainer(servingDeployments, comp.GetSpec().GetConfig()))
		transformers = append(transformers, ExtensionDeploymentOverrides(comp.GetSpec().GetWorkloadOverrides(), servingDeployments))
	}
	return transformers
}

func GetServingMonitoringPlatformManifests(ks base.KComponent) ([]mf.Manifest, error) {
	rbacManifest, err := getRBACManifest()
	if err != nil {
		return nil, err
	}

	// Serving has one sa for the control plane and one for the data plane, both need to be able to
	// authenticate requests for monitoring via kube rbac proxy
	for _, sa := range []string{"controller", "activator"} {
		rbM, err := CreateClusterRoleBindingManifest(sa, ks.GetNamespace())
		if err != nil {
			return nil, err
		}
		rbacManifest = rbacManifest.Append(*rbM)
	}

	for c := range servingDeployments {
		if err := AppendManifestsForComponent(c, ks.GetNamespace(), &rbacManifest); err != nil {
			return nil, err
		}
	}
	return []mf.Manifest{rbacManifest}, nil
}
