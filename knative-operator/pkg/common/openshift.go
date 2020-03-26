package common

import (
	"context"
	"fmt"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = Log

func Mutate(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	stages := []func(*servingv1alpha1.KnativeServing, client.Client) error{
		ingressClass,
		ingress,
		configureLogURLTemplate,
		ensureCustomCerts,
		imagesFromEnviron,
		defaultToHa,
	}
	for _, stage := range stages {
		if err := stage(ks, c); err != nil {
			return fmt.Errorf("failed to mutate KnativeServing: %w", err)
		}
	}
	return nil
}

func defaultToHa(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	if ks.Spec.HighAvailability == nil {
		ks.Spec.HighAvailability = &servingv1alpha1.HighAvailability{
			Replicas: 2,
		}
	}

	return nil
}

func ingressClass(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	Configure(ks, "network", "ingress.class", "kourier.ingress.networking.knative.dev")
	return nil
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
func ensureCustomCerts(ks *servingv1alpha1.KnativeServing, _ client.Client) error {
	if ks.Spec.ControllerCustomCerts == (servingv1alpha1.CustomCerts{}) {
		ks.Spec.ControllerCustomCerts = servingv1alpha1.CustomCerts{
			Name: "config-service-ca",
			Type: "ConfigMap",
		}
	}
	log.Info("ControllerCustomCerts", "certs", ks.Spec.ControllerCustomCerts)
	return nil
}

// imagesFromEnviron overrides registry images
func imagesFromEnviron(ks *servingv1alpha1.KnativeServing, _ client.Client) error {
	ks.Spec.Registry.Override = buildImageOverrideMapFromEnviron()

	if defaultVal, ok := ks.Spec.Registry.Override["default"]; ok {
		ks.Spec.Registry.Default = defaultVal
	}

	// special case for queue-proxy
	if qpVal, ok := ks.Spec.Registry.Override["queue-proxy"]; ok {
		Configure(ks, "deployment", "queueSidecarImage", qpVal)
	}
	log.Info("Setting", "registry", ks.Spec.Registry)
	return nil
}
