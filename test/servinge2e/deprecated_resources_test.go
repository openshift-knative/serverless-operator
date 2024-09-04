package servinge2e

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-knative/serverless-operator/test"
)

// DomainMapping functionality is merged into Serving controller.
// All related standalone resources with "domainmapping-*" prefix should be removed.
func TestDomainMappingResource(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	for _, dep := range []string{"domain-mapping", "domainmapping-webhook"} {
		t.Run("Deployments "+dep, func(t *testing.T) {
			if _, err := ctx.Clients.Kube.AppsV1().Deployments(test.ServingNamespace).Get(context.Background(), dep, metav1.GetOptions{}); err != nil {
				if !apierrors.IsNotFound(err) {
					t.Fatalf("Failed to verify deployment is deprecated & removed %s: %v", dep, err)
				}
			}
		})
	}
	t.Run("Service", func(t *testing.T) {
		if _, err := ctx.Clients.Kube.CoreV1().Services(test.ServingNamespace).Get(context.Background(), "domainmapping-webhook", metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				t.Fatalf("Failed to verify service is deprecated & removed %s: %v", "domainmapping-webhook", err)
			}
		}
	})
	t.Run("Secret", func(t *testing.T) {
		if _, err := ctx.Clients.Kube.CoreV1().Secrets(test.ServingNamespace).Get(context.Background(), "domainmapping-webhook-certs", metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				t.Fatalf("Failed to verify secret is deprecated & removed %s: %v", "domainmapping-webhook-certs", err)
			}
		}
	})
	t.Run("Secret", func(t *testing.T) {
		if _, err := ctx.Clients.Kube.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), "webhook.domainmapping.serving.knative.dev", metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				t.Fatalf("Failed to verify mutating webhook is deprecated & removed %s: %v", "webhook.domainmapping.serving.knative.dev", err)
			}
		}
	})
	t.Run("Secret", func(t *testing.T) {
		if _, err := ctx.Clients.Kube.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), "validation.webhook.domainmapping.serving.knative.dev", metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				t.Fatalf("Failed to verify validating webhook is deprecated & removed %s: %v", "webhook.domainmapping.serving.knative.dev", err)
			}
		}
	})
}

// SRVKS-1264 - cleanup of old internal TLS secrets
func TestOldTLSSecret(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	t.Run("Secret", func(t *testing.T) {
		if _, err := ctx.Clients.Kube.CoreV1().Secrets(test.ServingNamespace).Get(context.Background(), "control-serving-certs", metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				t.Fatalf("Failed to verify secret is deprecated & removed %s: %v", "control-serving-certs", err)
			}
		}
	})
}
