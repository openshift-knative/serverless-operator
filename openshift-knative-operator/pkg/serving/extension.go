package serving

import (
	"context"
	"fmt"
	"os"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"

	"github.com/openshift-knative/serverless-operator/pkg/client/clientset/versioned"
	ocpclient "github.com/openshift-knative/serverless-operator/pkg/client/injection/client"
	operator "knative.dev/operator/pkg/reconciler/common"
)

const loggingURLTemplate = "https://%s/app/kibana#/discover?_a=(index:.all,query:'kubernetes.labels.serving_knative_dev%%5C%%2FrevisionUID:${REVISION_UID}')"

// NewExtension creates a new extension for a Knative Serving controller.
func NewExtension(ctx context.Context) operator.Extension {
	return &extension{
		ocpclient: ocpclient.Get(ctx),
	}
}

type extension struct {
	ocpclient versioned.Interface
}

func (e *extension) Transformers(v1alpha1.KComponent) []mf.Transformer {
	return nil
}

func (e *extension) Reconcile(ctx context.Context, comp v1alpha1.KComponent) error {
	ks := comp.(*v1alpha1.KnativeServing)

	// Mark the Kourier dependency as installing to avoid race conditions with readiness.
	if ks.Status.GetCondition(v1alpha1.DependenciesInstalled).IsUnknown() {
		ks.Status.MarkDependencyInstalling("Kourier")
	}

	// Set the default host to the cluster's host.
	if domain, err := e.fetchClusterHost(ctx); err != nil {
		return fmt.Errorf("failed to fetch cluster host: %w", err)
	} else if domain != "" {
		common.Configure(&ks.Spec.CommonSpec, "domain", domain, "")
	}

	// Attempt to locate kibana route which is available if openshift-logging has been configured
	if loggingHost := e.fetchLoggingHost(ctx); loggingHost != "" {
		common.Configure(&ks.Spec.CommonSpec, "observability", "logging.revision-url-template",
			fmt.Sprintf(loggingURLTemplate, loggingHost))
	}

	// Override images.
	// TODO(SRVCOM-1069): Rethink overriding behavior and/or error surfacing.
	images := common.ImageMapFromEnvironment(os.Environ())
	ks.Spec.Registry.Override = common.ImageMapFromEnvironment(os.Environ())
	ks.Spec.Registry.Default = images["default"]
	common.Configure(&ks.Spec.CommonSpec, "deployment", "queueSidecarImage", images["queue-proxy"])

	// Default to 2 replicas.
	if ks.Spec.HighAvailability == nil {
		ks.Spec.HighAvailability = &v1alpha1.HighAvailability{
			Replicas: 2,
		}
	}

	// Use Kourier.
	// TODO(SRVCOM-1069): Rethink overriding behavior and/or error surfacing.
	common.Configure(&ks.Spec.CommonSpec, "network", "ingress.class", "kourier.ingress.networking.knative.dev")

	// Override the default domainTemplate to use $name-$ns rather than $name.$ns.
	// TODO(SRVCOM-1069): Rethink overriding behavior and/or error surfacing.
	common.Configure(&ks.Spec.CommonSpec, "network", "domainTemplate", "{{.Name}}-{{.Namespace}}.{{.Domain}}")

	// Ensure webhook has 1G of memory.
	common.EnsureContainerMemoryLimit(&ks.Spec.CommonSpec, "webhook", resource.MustParse("1024Mi"))

	// Add custom-certificates to the deployments (ConfigMap creation remains in the old
	// operator for now)
	if ks.Spec.ControllerCustomCerts == (v1alpha1.CustomCerts{}) {
		ks.Spec.ControllerCustomCerts = v1alpha1.CustomCerts{
			Name: "config-service-ca",
			Type: "ConfigMap",
		}
	}

	return nil
}

func (e *extension) Finalize(context.Context, v1alpha1.KComponent) error {
	return nil
}

// fetchClusterHost fetches the cluster's hostname from the cluster's ingress config.
func (e *extension) fetchClusterHost(ctx context.Context) (string, error) {
	ingress, err := e.ocpclient.ConfigV1().Ingresses().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to fetch cluster config: %w", err)
	}
	return ingress.Spec.Domain, nil
}

// fetchLoggingHost fetches the hostname of the Kibana installed by Openshift Logging,
// if present.
func (e *extension) fetchLoggingHost(ctx context.Context) string {
	route, err := e.ocpclient.RouteV1().Routes("openshift-logging").Get(ctx, "kibana", metav1.GetOptions{})
	if err != nil || len(route.Status.Ingress) == 0 {
		return ""
	}
	return route.Status.Ingress[0].Host
}
