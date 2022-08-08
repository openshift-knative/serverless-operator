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
	networkCMName           = "network"

	// TODO: Use "knative.dev/networking/pkg/config" once the repo pulled Knative 1.6.
	// Backport messes up the dependencies.
	InternalEncryptionKey = "internal-encryption"
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

func addKourierEnvValues(ks base.KComponent) mf.Transformer {
	envVars := []corev1.EnvVar{
		{Name: "KOURIER_HTTPOPTION_DISABLED", Value: "true"},
		{Name: "SERVING_NAMESPACE", Value: "knative-serving"},
	}
	if ks.GetSpec().GetConfig()[networkCMName][InternalEncryptionKey] != "" {
		envVars = append(envVars,
			corev1.EnvVar{Name: "CERTS_SECRET_NAMESPACE", Value: "openshift-ingress"},
			corev1.EnvVar{Name: "CERTS_SECRET_NAME", Value: "router-certs-default"})
	}
	return common.InjectEnvironmentIntoDeployment("net-kourier-controller", "controller", envVars...)
}
