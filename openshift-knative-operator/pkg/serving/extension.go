package serving

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver/v4"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	"github.com/openshift-knative/serverless-operator/pkg/client/clientset/versioned"
	ocpclient "github.com/openshift-knative/serverless-operator/pkg/client/injection/client"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	operator "knative.dev/operator/pkg/reconciler/common"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	deploymentinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

const (
	loggingURLTemplate = "https://%s/app/kibana#/discover?_a=(index:.all,query:'kubernetes.labels.serving_knative_dev%%5C%%2FrevisionUID:${REVISION_UID}')"
	requiredNsEnvName  = "REQUIRED_SERVING_NAMESPACE"

	defaultDomainTemplate = "{{.Name}}-{{.Namespace}}.{{.Domain}}"
)

// NewExtension creates a new extension for a Knative Serving controller.
func NewExtension(ctx context.Context, impl *controller.Impl) operator.Extension {
	deploymentInformer := deploymentinformer.Get(ctx)

	// We move the Kourier deployments into a different namespace so the usual informer
	// that enqueues the OwnerRef doesn't catch those, so we add them here explicitly.
	deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: reconciler.LabelExistsFilterFunc(socommon.ServingOwnerNamespace),
		Handler:    controller.HandleAll(impl.EnqueueLabelOfNamespaceScopedResource(socommon.ServingOwnerNamespace, socommon.ServingOwnerName)),
	})

	return &extension{
		ocpclient:  ocpclient.Get(ctx),
		kubeclient: kubeclient.Get(ctx),
	}
}

type extension struct {
	ocpclient  versioned.Interface
	kubeclient kubernetes.Interface
}

func (e *extension) Manifests(ks base.KComponent) ([]mf.Manifest, error) {
	return monitoring.GetServingMonitoringPlatformManifests(ks)
}

func (e *extension) Transformers(ks base.KComponent) []mf.Transformer {
	return append([]mf.Transformer{
		common.InjectCommonLabelIntoNamespace(),
		common.InjectEnvironmentIntoDeployment("controller", "controller",
			corev1.EnvVar{Name: "HTTP_PROXY", Value: os.Getenv("HTTP_PROXY")},
			corev1.EnvVar{Name: "HTTPS_PROXY", Value: os.Getenv("HTTPS_PROXY")},
			corev1.EnvVar{Name: "NO_PROXY", Value: os.Getenv("NO_PROXY")},
		),
		overrideKourierNamespace(ks),
		addHTTPOptionDisabledEnvValue(),
	}, monitoring.GetServingTransformers(ks)...)
}

func (e *extension) Reconcile(ctx context.Context, comp base.KComponent) error {
	ks := comp.(*operatorv1alpha1.KnativeServing)
	log := logging.FromContext(ctx)

	// Make sure Knative Serving is always installed in the defined namespace.
	requiredNs := os.Getenv(requiredNsEnvName)
	if requiredNs != "" && ks.Namespace != requiredNs {
		ks.Status.MarkInstallFailed(fmt.Sprintf("Knative Serving must be installed into the namespace %q", requiredNs))
		return controller.NewPermanentError(fmt.Errorf("deployed Knative Serving into unsupported namespace %q", ks.Namespace))
	}

	// Mark failed dependencies as succeeded since we're no longer using that mechanism anyway.
	if ks.Status.GetCondition(base.DependenciesInstalled).IsFalse() {
		ks.Status.MarkDependenciesInstalled()
	}

	// Set the default host to the cluster's host.
	if domain, err := e.fetchClusterHost(ctx); err != nil {
		return fmt.Errorf("failed to fetch cluster host: %w", err)
	} else if domain != "" {
		common.Configure(&ks.Spec.CommonSpec, "domain", domain, "")
	}

	// Attempt to locate kibana route which is available if openshift-logging has been configured
	if loggingHost := e.fetchLoggingHost(ctx); loggingHost != "" {
		common.Configure(&ks.Spec.CommonSpec, monitoring.ObservabilityCMName, "logging.revision-url-template",
			fmt.Sprintf(loggingURLTemplate, loggingHost))
	}

	// Override images.
	// TODO(SRVCOM-1069): Rethink overriding behavior and/or error surfacing.
	images := common.ImageMapFromEnvironment(os.Environ())
	ks.Spec.Registry.Override = images
	ks.Spec.Registry.Default = images["default"]
	common.Configure(&ks.Spec.CommonSpec, "deployment", "queueSidecarImage", images["queue-proxy"])

	// Default to 2 replicas.
	if ks.Spec.HighAvailability == nil {
		ks.Spec.HighAvailability = &base.HighAvailability{
			Replicas: 2,
		}
	}

	// Apply an Ingress config with Kourier enabled if nothing else is defined.
	defaultToKourier(ks)
	common.ConfigureIfUnset(&ks.Spec.CommonSpec, "network", "ingress.class", defaultIngressClass(ks))

	// Changing service type from LoadBalancer to ClusterIP has a bug https://github.com/kubernetes/kubernetes/pull/95196
	// Do not apply the default if the version is less than v1.20.0.
	if err := checkMinimumVersion(e.kubeclient.Discovery(), "1.20.0"); err != nil {
		log.Warnf("Could not apply default service type for Kourier Gateway: %v", err)
	} else {
		// Apply Kourier gateway service type.
		defaultKourierServiceType(ks)
	}

	// Override the default domainTemplate to use $name-$ns rather than $name.$ns.
	common.ConfigureIfUnset(&ks.Spec.CommonSpec, "network", "domainTemplate", defaultDomainTemplate)

	// Default the URL scheme to HTTPS if nothing else is defined.
	common.ConfigureIfUnset(&ks.Spec.CommonSpec, "network", "defaultExternalScheme", "https")

	// Ensure webhook has 1G of memory.
	common.EnsureContainerMemoryLimit(&ks.Spec.CommonSpec, "webhook", resource.MustParse("1024Mi"))

	// Add custom-certificates to the deployments (ConfigMap creation remains in the old
	// operator for now)
	if ks.Spec.ControllerCustomCerts == (base.CustomCerts{}) {
		ks.Spec.ControllerCustomCerts = base.CustomCerts{
			Name: "config-service-ca",
			Type: "ConfigMap",
		}
	}

	// Explicitly set autocreateClusterDomainClaims to true if not otherwise set to be
	// independent from upstream default changes.
	common.ConfigureIfUnset(&ks.Spec.CommonSpec, "network", "autocreateClusterDomainClaims", "true")

	// Temporary fix for SRVKS-743
	if ks.Spec.Ingress.Istio.Enabled {
		common.ConfigureIfUnset(&ks.Spec.CommonSpec, monitoring.ObservabilityCMName, monitoring.ObservabilityBackendKey, "none")
	}

	return monitoring.ReconcileMonitoringForServing(ctx, e.kubeclient, ks)
}

func (e *extension) Finalize(ctx context.Context, comp base.KComponent) error {
	ks := comp.(*operatorv1alpha1.KnativeServing)

	// Delete the ingress namespaces manually. Manifestival won't do it for us in upgrade cases.
	// See: https://github.com/manifestival/manifestival/issues/85
	err := e.kubeclient.CoreV1().Namespaces().Delete(ctx, kourierNamespace(ks.GetNamespace()), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to remove ingress namespace: %w", err)
	}

	// Also default to Kourier here to pick the right manifest to uninstall.
	defaultToKourier(ks)

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

// checkMinimumVersion checks if the version in the arg meets the requirement or not.
// It is similar logic with CheckMinimumVersion() in knative.dev/pkg/version.
func checkMinimumVersion(versioner discovery.ServerVersionInterface, version string) error {
	v, err := versioner.ServerVersion()
	if err != nil {
		return err
	}
	currentVersion, err := semver.Make(normalizeVersion(v.GitVersion))
	if err != nil {
		return err
	}

	minimumVersion, err := semver.Make(normalizeVersion(version))
	if err != nil {
		return err
	}

	// If no specific pre-release requirement is set, we default to "-0" to always allow
	// pre-release versions of the same Major.Minor.Patch version.
	if len(minimumVersion.Pre) == 0 {
		minimumVersion.Pre = []semver.PRVersion{{VersionNum: 0, IsNum: true}}
	}

	if currentVersion.LT(minimumVersion) {
		return fmt.Errorf("kubernetes version %q is not compatible, need at least %q",
			currentVersion, minimumVersion)
	}
	return nil
}

func normalizeVersion(v string) string {
	if strings.HasPrefix(v, "v") {
		// No need to account for unicode widths.
		return v[1:]
	}
	return v
}
