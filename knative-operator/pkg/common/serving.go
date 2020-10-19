package common

import (
	"context"
	"fmt"
	"os"

	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = Log

const (
	// defaultDomainTemplate is a value for domainTemplate in config-network.
	// As Knative on OpenShift uses OpenShift's wildcard cert the domain level must have "<sub>.domain", not "<sub1>.<sub2>.domain".
	DefaultDomainTemplate = "{{.Name}}-{{.Namespace}}.{{.Domain}}"

	// defaultIngressClass is a value for ingress.class in config-network.
	DefaultIngressClass = "kourier.ingress.networking.knative.dev"
)

func Mutate(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	if err := ingress(ks, c); err != nil {
		return fmt.Errorf("failed to configure ingress: %w", err)
	}

	if err := configureLogURLTemplate(ks, c); err != nil {
		return fmt.Errorf("failed to configure log URL: %w", err)
	}

	domainTemplate(ks)
	ingressClass(ks)
	ensureCustomCerts(ks)
	imagesFromEnviron(ks)
	ensureServingWebhookMemoryLimit(ks)
	defaultToHa(ks)
	return nil
}

func defaultToHa(ks *servingv1alpha1.KnativeServing) {
	if ks.Spec.HighAvailability == nil {
		ks.Spec.HighAvailability = &servingv1alpha1.HighAvailability{
			Replicas: 2,
		}
	}
}

func ensureServingWebhookMemoryLimit(ks *servingv1alpha1.KnativeServing) {
	EnsureContainerMemoryLimit(&ks.Spec.CommonSpec, "webhook", resource.MustParse("1024Mi"))
}

func domainTemplate(ks *servingv1alpha1.KnativeServing) {
	Configure(ks, "network", "domainTemplate", DefaultDomainTemplate)
}

func ingressClass(ks *servingv1alpha1.KnativeServing) {
	Configure(ks, "network", "ingress.class", DefaultIngressClass)
}

// configure ingress
func ingress(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	ingressConfig := &configv1.Ingress{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: "cluster"}, ingressConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return fmt.Errorf("failed to fetch ingress config: %w", err)
		}
		log.Info("No OpenShift ingress config available")
		return nil
	}
	domain := ingressConfig.Spec.Domain
	if len(domain) > 0 {
		Configure(ks, "domain", domain, "")
	}
	return nil
}

// configure observability if ClusterLogging is installed
func configureLogURLTemplate(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	const (
		configmap = "observability"
		key       = "logging.revision-url-template"
		name      = "kibana"
		namespace = "openshift-logging"
	)
	// attempt to locate kibana route which is available if openshift-logging has been configured
	route := &routev1.Route{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: namespace}, route); err != nil {
		log.Info(fmt.Sprintf("No revision-url-template; no route for %s/%s found", namespace, name))
		return nil
	}
	// retrieve host from kibana route, construct a concrete logUrl template with actual host name, update observability
	if len(route.Status.Ingress) > 0 {
		host := route.Status.Ingress[0].Host
		if host != "" {
			url := "https://" + host + "/app/kibana#/discover?_a=(index:.all,query:'kubernetes.labels.serving_knative_dev%5C%2FrevisionUID:${REVISION_UID}')"
			Configure(ks, configmap, key, url)
		}
	}
	return nil
}

// configure controller with custom certs for openshift registry if
// not already set
func ensureCustomCerts(ks *servingv1alpha1.KnativeServing) {
	if ks.Spec.ControllerCustomCerts == (servingv1alpha1.CustomCerts{}) {
		ks.Spec.ControllerCustomCerts = servingv1alpha1.CustomCerts{
			Name: "config-service-ca",
			Type: "ConfigMap",
		}
	}
	log.Info("ControllerCustomCerts", "certs", ks.Spec.ControllerCustomCerts)
}

// imagesFromEnviron overrides registry images
func imagesFromEnviron(ks *servingv1alpha1.KnativeServing) {
	ks.Spec.Registry.Override = BuildImageOverrideMapFromEnviron(os.Environ(), "SERVING")

	if defaultVal, ok := ks.Spec.Registry.Override["default"]; ok {
		ks.Spec.Registry.Default = defaultVal
	}

	// special case for queue-proxy
	if qpVal, ok := ks.Spec.Registry.Override["queue-proxy"]; ok {
		Configure(ks, "deployment", "queueSidecarImage", qpVal)
	}
	log.Info("Setting", "registry", ks.Spec.Registry)
}
