package common

import (
	"os"

	eventingv1alpha1 "github.com/knative-sandbox/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MutateEventing(ke *eventingv1alpha1.KnativeEventing, c client.Client) error {
	stages := []func(*eventingv1alpha1.KnativeEventing, client.Client) error{
		eventingImagesFromEnviron,
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
