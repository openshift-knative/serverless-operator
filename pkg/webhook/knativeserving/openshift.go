package knativeserving

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/openshift-knative/knative-serving-openshift/pkg"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = pkg.Log.WithName("webhook")

// configure egress
func (a *KnativeServingConfigurator) egress(ctx context.Context, ks *servingv1alpha1.KnativeServing) error {
	networkConfig := &configv1.Network{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: "cluster"}, networkConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return err
		}
		log.Info("No OpenShift cluster network config available")
		return nil
	}
	network := strings.Join(networkConfig.Spec.ServiceNetwork, ",")
	pkg.Configure(ks, "network", "istio.sidecar.includeOutboundIPRanges", network)
	return nil
}

// configure ingress
func (a *KnativeServingConfigurator) ingress(ctx context.Context, ks *servingv1alpha1.KnativeServing) error {
	ingressConfig := &configv1.Ingress{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return err
		}
		log.Info("No OpenShift ingress config available")
		return nil
	}
	domain := ingressConfig.Spec.Domain
	if len(domain) > 0 {
		pkg.Configure(ks, "domain", domain, "")
	}
	return nil
}

// configure observability if ClusterLogging is installed
func (a *KnativeServingConfigurator) configureLogURLTemplate(ctx context.Context, ks *servingv1alpha1.KnativeServing) error {
	const (
		configmap = "observability"
		key       = "logging.revision-url-template"
		name      = "kibana"
		namespace = "openshift-logging"
	)
	// attempt to locate kibana route which is available if openshift-logging has been configured
	route := &routev1.Route{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, route); err != nil {
		log.Info(fmt.Sprintf("No revision-url-template; no route for %s/%s found", namespace, name))
		return nil
	}
	// retrieve host from kibana route, construct a concrete logUrl template with actual host name, update observability
	if len(route.Status.Ingress) > 0 {
		host := route.Status.Ingress[0].Host
		if host != "" {
			url := "https://" + host + "/app/kibana#/discover?_a=(index:.all,query:'kubernetes.labels.serving_knative_dev%5C%2FrevisionUID:${REVISION_UID}')"
			pkg.Configure(ks, configmap, key, url)
		}
	}
	return nil
}

// configure controller with custom certs for openshift registry if
// not already set
func (a *KnativeServingConfigurator) ensureCustomCerts(ctx context.Context, instance *servingv1alpha1.KnativeServing) error {
	if instance.Spec.ControllerCustomCerts == (servingv1alpha1.CustomCerts{}) {
		instance.Spec.ControllerCustomCerts = servingv1alpha1.CustomCerts{
			Name: "config-service-ca",
			Type: "ConfigMap",
		}
	}
	log.Info("ControllerCustomCerts", "certs", instance.Spec.ControllerCustomCerts)
	return nil
}

// imagesFromEnviron overrides registry images
func (a *KnativeServingConfigurator) imagesFromEnviron(ctx context.Context, instance *servingv1alpha1.KnativeServing) error {
	if instance.Spec.Registry.Override == nil {
		instance.Spec.Registry.Override = map[string]string{}
	} // else return since overriding user from env might surprise me?
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], "IMAGE_") {
			name := strings.SplitN(pair[0], "_", 2)[1]
			switch name {
			case "default":
				instance.Spec.Registry.Default = pair[1]
			case "queue-proxy":
				pkg.Configure(instance, "deployment", "queueSidecarImage", pair[1])
				fallthrough
			default:
				instance.Spec.Registry.Override[name] = pair[1]
			}
		}
	}
	log.Info("Setting", "registry", instance.Spec.Registry)
	return nil
}

// validate minimum openshift version
func (v *KnativeServingValidator) validateVersion(ctx context.Context, ks *servingv1alpha1.KnativeServing) (bool, string, error) {
	minVersion, err := semver.NewVersion(os.Getenv("MIN_OPENSHIFT_VERSION"))
	if err != nil {
		return false, "Unable to validate version; check MIN_OPENSHIFT_VERSION env var", nil
	}

	clusterVersion := &configv1.ClusterVersion{}
	if err := v.client.Get(ctx, client.ObjectKey{Name: "version"}, clusterVersion); err != nil {
		return false, "Unable to get ClusterVersion", err
	}

	current, err := semver.NewVersion(clusterVersion.Status.Desired.Version)
	if err != nil {
		return false, "Could not parse version string", err
	}

	if current.Major == 0 && current.Minor == 0 {
		return true, "CI build detected, bypassing version check", nil
	}

	if current.LessThan(*minVersion) {
		msg := fmt.Sprintf("Version constraint not fulfilled: minimum version: %s, current version: %s", minVersion.String(), current.String())
		return false, msg, nil
	}
	return true, "", nil
}

// validate required namespace, if any
func (v *KnativeServingValidator) validateNamespace(ctx context.Context, ks *servingv1alpha1.KnativeServing) (bool, string, error) {
	ns, required := os.LookupEnv("REQUIRED_NAMESPACE")
	if required && ns != ks.Namespace {
		return false, fmt.Sprintf("KnativeServing may only be created in %s namespace", ns), nil
	}
	return true, "", nil
}
