package serving

import (
	"context"
	"os"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"k8s.io/apimachinery/pkg/api/resource"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	operator "knative.dev/operator/pkg/reconciler/common"
)

// NewExtension creates a new extension for a Knative Serving controller.
func NewExtension(ctx context.Context) operator.Extension {
	return &extension{}
}

type extension struct{}

func (e *extension) Transformers(v1alpha1.KComponent) []mf.Transformer {
	return nil
}

func (e *extension) Reconcile(ctx context.Context, comp v1alpha1.KComponent) error {
	ks := comp.(*v1alpha1.KnativeServing)

	// Override images.
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
	common.Configure(&ks.Spec.CommonSpec, "network", "ingress.class", "kourier.ingress.networking.knative.dev")

	// Override the default domainTemplate to use $name-$ns rather than $name.$ns.
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
