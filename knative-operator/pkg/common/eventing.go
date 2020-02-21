package common

import (
	eventingv1alpha1 "knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MutateEventing(ke *eventingv1alpha1.KnativeEventing, c client.Client) error {
	stages := []func(*eventingv1alpha1.KnativeEventing, client.Client) error{
		eventingImagesFromEnviron,
		annotateTimestampEventing,
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
	updateImagesFromEnviron((*servingv1alpha1.Registry)(&ke.Spec.Registry))
	return nil
}

// Mark the time when instance configured for OpenShift
func annotateTimestampEventing(ke *eventingv1alpha1.KnativeEventing, _ client.Client) error {
	if ke.GetAnnotations() == nil {
		ke.SetAnnotations(map[string]string{})
	}
	annotateTimestamp(ke.GetAnnotations())

	return nil
}
