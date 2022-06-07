package serving

import (
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/base"
)

const (
	providerLabel           = "networking.knative.dev/ingress-provider"
	kourierIngressClassName = "kourier.ingress.networking.knative.dev"
)

// overrideKourierNamespace overrides the namespace of all Kourier related resources to
// the -ingress suffix to be backwards compatible.
func overrideKourierNamespace(ks base.KComponent) mf.Transformer {
	nsInjector := mf.InjectNamespace(kourierNamespace(ks.GetNamespace()))
	return func(u *unstructured.Unstructured) error {
		provider := u.GetLabels()[providerLabel]
		if provider != "kourier" {
			return nil
		}

		labels := u.GetLabels()
		if labels == nil {
			labels = make(map[string]string, 2)
		}
		labels[socommon.ServingOwnerNamespace] = ks.GetNamespace()
		labels[socommon.ServingOwnerName] = ks.GetName()
		u.SetLabels(labels)

		// We need to unset OwnerReferences so Openshift doesn't delete Kourier ressources.
		u.SetOwnerReferences(nil)
		return nsInjector(u)
	}
}

// kourierNamespace returns the namespace Kourier was installed into for backwards
// compatibility.
func kourierNamespace(servingNs string) string {
	return servingNs + "-ingress"
}

func addHTTPOptionDisabledEnvValue() mf.Transformer {
	return common.InjectEnvironmentIntoDeployment("net-kourier-controller", "controller",
		corev1.EnvVar{Name: "KOURIER_HTTPOPTION_DISABLED", Value: "true"},
	)
}
