package encryption

import (
	"context"
	"fmt"
	"testing"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress/resources"
	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/networking/pkg/apis/networking"
	"knative.dev/networking/pkg/certificates"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/serving/pkg/apis/serving"
)

func TestExternalAccessWithEncryptionEnabled(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
	defer test.CleanupAll(t, ctx)

	ksvc := test.Service("encryption-test", test.Namespace, pkgTest.ImagePath(test.HelloworldGoImg), nil, nil)
	ksvc = test.WithServiceReadyOrFail(ctx, ksvc)

	// Check if the service is reachable.
	servinge2e.WaitForRouteServingText(t, ctx, ksvc.Status.URL.URL(), servinge2e.HelloworldText)

	// Verify the OCP route the operator created.
	routes, err := ctx.Clients.Route.Routes(test.IngressNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", resources.OpenShiftIngressLabelKey, ksvc.Name),
	})
	if err != nil {
		t.Fatalf("Failed to list routes: %v", err)
	}
	for _, r := range routes.Items {
		if r.ObjectMeta.Labels[resources.OpenShiftIngressLabelKey] == ksvc.Name {
			if !(r.Spec.TLS != nil && r.Spec.TLS.Termination == routev1.TLSTerminationPassthrough && r.Spec.TLS.InsecureEdgeTerminationPolicy == routev1.InsecureEdgeTerminationPolicyRedirect) {
				t.Fatalf("Route %s does not have expected TLS termination and http redirects: %v", r.Name, r.Spec.TLS)
			}
			if r.Spec.Port.TargetPort.StrVal != "https" {
				t.Fatalf("Route %s does not have expected https target port: %v", r.Name, r.Spec.Port.TargetPort)
			}
		}
	}
}

func TestClusterLocalAccessWithEncryptionEnabled(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
	defer test.CleanupAll(t, ctx)

	ksvc := test.Service("encryption-test", test.Namespace, pkgTest.ImagePath(test.HelloworldGoImg), nil, nil)
	ksvc.ObjectMeta.Labels = map[string]string{networking.VisibilityLabelKey: serving.VisibilityClusterLocal}
	ksvc = test.WithServiceReadyOrFail(ctx, ksvc)

	// Get cert-manager CA cert that signed the cluster-local-domain certs
	ca, err := getCertManagerCA(ctx.Clients)
	if err != nil {
		t.Fatalf("Could not get cert-manager CA: %v", err)
	}

	// Check if the service is reachable with https on cluster-local
	httpProxy := test.WithServiceReadyOrFail(ctx, servinge2e.HTTPProxyService(ksvc.Name+"-proxy", test.Namespace,
		"", ksvc.Status.URL.Host, string(ca.Data[certificates.CertName]), nil, nil))

	servinge2e.WaitForRouteServingText(t, ctx, httpProxy.Status.URL.URL(), servinge2e.HelloworldText)
}
