package common

import (
	eventingv1alpha1 "knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
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
	ke.Spec.Registry.Override = buildImageOverrideMapFromEnviron()

	if defaultVal, ok := ke.Spec.Registry.Override["default"]; ok {
		ke.Spec.Registry.Default = defaultVal
	}

	log.Info("Setting", "registry", ke.Spec.Registry)
	return nil
}
