package common

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"os"

	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MutateEventing(ke *eventingv1alpha1.KnativeEventing, c client.Client) error {
	stages := []func(*eventingv1alpha1.KnativeEventing, client.Client) error{
		eventingImagesFromEnviron,
		ensureEventingWebhookMemoryLimit,
	}
	for _, stage := range stages {
		if err := stage(ke, c); err != nil {
			return err
		}
	}
	return nil
}

// eventingImagesFromEnviron overrides registry images
func eventingImagesFromEnviron(ke *eventingv1alpha1.KnativeEventing, _ client.Client) error {
	ke.Spec.Registry.Override = BuildImageOverrideMapFromEnviron(os.Environ())

	if defaultVal, ok := ke.Spec.Registry.Override["default"]; ok {
		ke.Spec.Registry.Default = defaultVal
	}

	log.Info("Setting", "registry", ke.Spec.Registry)
	return nil
}

func ensureEventingWebhookMemoryLimit(ks *eventingv1alpha1.KnativeEventing, c client.Client) error {
	return EnsureContainerMemoryLimit(&ks.Spec.CommonSpec, "eventing-webhook", resource.MustParse("1024Mi"))
}
