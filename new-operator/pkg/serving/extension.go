package serving

import (
	"context"
	"os"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	operator "knative.dev/operator/pkg/reconciler/common"

	"github.com/openshift-knative/serverless-operator/new-operator/pkg/client/clientset/versioned"
	ocpclient "github.com/openshift-knative/serverless-operator/new-operator/pkg/client/injection/client"
	"github.com/openshift-knative/serverless-operator/new-operator/pkg/common"
)

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
	// Silently apply our defaulting.
	ks := comp.(*v1alpha1.KnativeServing)

	// Fetch the proper domain.
	ingress, err := e.ocpclient.ConfigV1().Ingresses().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}
	domain := ingress.Spec.Domain
	if domain != "" {
		configure(ks, "domain", domain, "")
	}

	// Default to 2 replicas.
	if ks.Spec.HighAvailability == nil {
		ks.Spec.HighAvailability = &v1alpha1.HighAvailability{
			Replicas: 2,
		}
	}

	// Use kourier.
	configure(ks, "network", "ingress.class", "kourier.ingress.networking.knative.dev")
	configure(ks, "network", "domainTemplate", "{{.Name}}-{{.Namespace}}.{{.Domain}}")

	// Ensure custom certificates are used.
	if ks.Spec.ControllerCustomCerts == (v1alpha1.CustomCerts{}) {
		ks.Spec.ControllerCustomCerts = v1alpha1.CustomCerts{
			Name: "config-service-ca",
			Type: "ConfigMap",
		}
	}

	// Configure the correct image coordinates.
	configureImagesFromEnvironment(ks)

	// Ensure webhooks have 1G of memory.
	common.EnsureContainerMemoryLimit(&ks.Spec.CommonSpec, "webhook", resource.MustParse("1024Mi"))

	return nil
}

func (e *extension) Finalize(context.Context, v1alpha1.KComponent) error {
	return nil
}

// configure is a helper to set a value for a key, potentially overriding existing contents.
func configure(ks *v1alpha1.KnativeServing, cm, key, value string) {
	if ks.Spec.Config == nil {
		ks.Spec.Config = map[string]map[string]string{}
	}

	if ks.Spec.Config[cm] == nil {
		ks.Spec.Config[cm] = map[string]string{}
	}

	ks.Spec.Config[cm][key] = value
}

func configureImagesFromEnvironment(ks *v1alpha1.KnativeServing) {
	reg := ks.GetSpec().GetRegistry()

	reg.Override = common.ImageMapFromEnvironment(os.Environ())

	if defaultVal, ok := reg.Override["default"]; ok {
		reg.Default = defaultVal
	}

	if qpVal, ok := reg.Override["queue-proxy"]; ok {
		configure(ks, "deployment", "queueSidecarImage", qpVal)
	}
}
