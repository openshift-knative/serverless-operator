package serving

import (
	"context"
	"fmt"
	"strings"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const (
	providerLabel           = "networking.knative.dev/ingress-provider"
	kourierIngressClassName = "kourier.ingress.networking.knative.dev"
)

// overrideKourierNamespace overrides the namespace of all Kourier related resources to
// the -ingress suffix to be backwards compatible.
func overrideKourierNamespace(kourierNs string) mf.Transformer {
	nsInjector := mf.InjectNamespace(kourierNs)
	return func(u *unstructured.Unstructured) error {
		provider := u.GetLabels()[providerLabel]
		if provider != "kourier" {
			return nil
		}

		// We need to unset OwnerReferences so Openshift doesn't delete Kourier ressources.
		u.SetOwnerReferences(nil)
		return nsInjector(u)
	}
}

// replaceServiceSelector replaces the selector of the kourier-control service to the new
// selector after all components have successfully been rolled out.
// TODO: Remove after resources are bumped to 0.26
func replaceServiceSelector(ks v1alpha1.KComponent) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if !ks.GetStatus().IsReady() || !strings.Contains(ks.GetStatus().GetVersion(), "0.25.") {
			// Do nothing while we're not completely rolled out yet.
			return nil
		}

		if u.GetKind() != "Service" || u.GetName() != "kourier-control" {
			return nil
		}

		svc := &corev1.Service{}
		if err := scheme.Scheme.Convert(u, svc, nil); err != nil {
			return err
		}

		svc.Spec.Selector = map[string]string{
			"app": "net-kourier-controller",
		}

		return scheme.Scheme.Convert(svc, u, true)
	}
}

// removeObsoleteResources removes all resources that couldn't automatically be cleaned up
// due to renaming.
// TODO: Remove after resources are bumped to 0.26
func removeObsoleteResources(ctx context.Context, kubeclient kubernetes.Interface, ks v1alpha1.KComponent) error {
	if !ks.GetStatus().IsReady() || !strings.Contains(ks.GetStatus().GetVersion(), "0.25.") {
		// Do nothing while we're not completely rolled out yet.
		return nil
	}

	ns := kourierNamespace(ks.GetNamespace())
	if err := kubeclient.AppsV1().Deployments(ns).Delete(ctx, "3scale-kourier-control", v1.DeleteOptions{}); !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete obsolete Deployment: %w", err)
	}
	if err := kubeclient.CoreV1().ServiceAccounts(ns).Delete(ctx, "3scale-kourier", v1.DeleteOptions{}); !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete obsolete ServiceAccount: %w", err)
	}
	if err := kubeclient.RbacV1().ClusterRoles().Delete(ctx, "3scale-kourier", v1.DeleteOptions{}); !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete obsolete ClusterRole: %w", err)
	}
	if err := kubeclient.RbacV1().ClusterRoleBindings().Delete(ctx, "3scale-kourier", v1.DeleteOptions{}); !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete obsolete ClusterRoleBinding: %w", err)
	}
	return nil
}

// kourierNamespace returns the namespace Kourier was installed into for backwards
// compatibility.
func kourierNamespace(servingNs string) string {
	return servingNs + "-ingress"
}
