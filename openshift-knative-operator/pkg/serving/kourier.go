package serving

import (
	"context"
	"fmt"
	"strings"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

var gatewayResources = map[string]string{
	"knative-serving":        "Namespace",
	"kourier":                "Service",
	"kourier-internal":       "Service",
	"3scale-kourier-gateway": "Deployment",
	"kourier-bootstrap":      "ConfigMap",
}

const (
	providerLabel           = "networking.knative.dev/ingress-provider"
	kourierIngressClassName = "kourier.ingress.networking.knative.dev"
)

// overrideKourierNamespace overrides the namespace of Kourier Gateway related resources to
// the -ingress suffix to be backwards compatible.
func overrideKourierNamespace(ks operatorv1alpha1.KComponent) mf.Transformer {
	nsInjector := mf.InjectNamespace(kourierNamespace(ks.GetNamespace()))
	return func(u *unstructured.Unstructured) error {
		provider := u.GetLabels()[providerLabel]
		if provider != "kourier" {
			return nil
		}

		if gatewayResources[u.GetName()] != u.GetKind() {
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

// removeObsoleteResources removes all resources that couldn't automatically be cleaned up
// due to namespace transformation.
// TODO: Remove after resources are bumped to 1.3. (TODO update to 1.4)
func removeObsoleteResources(ctx context.Context, kubeclient kubernetes.Interface, ks v1alpha1.KComponent) error {
	if !ks.GetStatus().IsReady() || !strings.Contains(ks.GetStatus().GetVersion(), "1.1.") { // TODO: Update to 1.2.
		// Do nothing while we're not completely rolled out yet.
		return nil
	}

	ns := kourierNamespace(ks.GetNamespace())

	if err := kubeclient.CoreV1().ConfigMaps(ns).Delete(ctx, "config-kourier", metav1.DeleteOptions{}); !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete obsolete ConfigMap: %w", err)
	}
	if err := kubeclient.CoreV1().ServiceAccounts(ns).Delete(ctx, "net-kourier", metav1.DeleteOptions{}); !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete obsolete ServiceAccount: %w", err)
	}
	if err := kubeclient.AppsV1().Deployments(ns).Delete(ctx, "net-kourier-controller", metav1.DeleteOptions{}); !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete obsolete Deployment: %w", err)
	}
	if err := kubeclient.CoreV1().Services(ns).Delete(ctx, "net-kourier-controller", metav1.DeleteOptions{}); !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete obsolete Service: %w", err)
	}
	return nil
}
