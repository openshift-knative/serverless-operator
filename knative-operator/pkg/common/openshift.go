package common

import (
	"context"
	"fmt"
	"os"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = Log

func Mutate(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	stages := []func(*servingv1alpha1.KnativeServing, client.Client) error{
		ingressClass,
		egress,
		ingress,
		configureLogURLTemplate,
		ensureCustomCerts,
		imagesFromEnviron,
	}
	for _, stage := range stages {
		if err := stage(ks, c); err != nil {
			return err
		}
	}
	return nil
}

func ingressClass(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	Configure(ks, "network", "ingress.class", "kourier.ingress.networking.knative.dev")
	return nil
}

// configure egress
func egress(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	networkConfig := &configv1.Network{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: "cluster"}, networkConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return err
		}
		log.Info("No OpenShift cluster network config available")
		return nil
	}
	network := strings.Join(networkConfig.Spec.ServiceNetwork, ",")
	Configure(ks, "network", "istio.sidecar.includeOutboundIPRanges", network)
	return nil
}

// configure ingress
func ingress(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	ingressConfig := &configv1.Ingress{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: "cluster"}, ingressConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return err
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

// Update updates Knative controller env to use cluster wide proxy information
func Update(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	var proxyEnv = map[string]string{
		"HTTP_PROXY": os.Getenv("HTTP_PROXY"),
		"NO_PROXY":   os.Getenv("NO_PROXY"),
	}
	deploy := &appsv1.Deployment{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: "controller", Namespace: ks.GetNamespace()}, deploy); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	for c := range deploy.Spec.Template.Spec.Containers {
		for k, v := range proxyEnv {
			// If value is not empty then update deployment controller with env
			if v != "" {
				deploy.Spec.Template.Spec.Containers[c].Env = appendUnique(deploy.Spec.Template.Spec.Containers[c].Env, k, v)
			} else {
				// If value is empty then remove those keys from deployment controller
				deploy.Spec.Template.Spec.Containers[c].Env = remove(deploy.Spec.Template.Spec.Containers[c].Env, k)
			}
		}
	}
	return c.Update(context.TODO(), deploy)
}

func remove(env []v1.EnvVar, key string) []v1.EnvVar {
	for i := range env {
		if env[i].Name == key {
			return append(env[:i], env[i+1:]...)
		}
	}
	return env
}

func appendUnique(orgEnv []v1.EnvVar, key, value string) []v1.EnvVar {
	for i := range orgEnv {
		if orgEnv[i].Name == key {
			if value == "" {
				return remove(orgEnv, key)
			}
			orgEnv[i].Value = value
			return orgEnv
		}
	}
	if value != "" {
		return append(orgEnv, v1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	return orgEnv
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
	if ks.Spec.Registry.Override == nil {
		ks.Spec.Registry.Override = map[string]string{}
	} // else return since overriding user from env might surprise me?
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], "IMAGE_") {
			name := strings.SplitN(pair[0], "_", 2)[1]
			switch name {
			case "default":
				ks.Spec.Registry.Default = pair[1]
			case "queue-proxy":
				Configure(ks, "deployment", "queueSidecarImage", pair[1])
				fallthrough
			default:
				ks.Spec.Registry.Override[name] = pair[1]
			}
		}
	}
	log.Info("Setting", "registry", ks.Spec.Registry)
	return nil
}
