package common

import (
	eventingv1alpha1 "knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func MutateEventing(ke *eventingv1alpha1.KnativeEventing, c client.Client) error {
	stages := []func(*eventingv1alpha1.KnativeEventing, client.Client) error{
		logEventing,
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
	if ke.Spec.Registry.Override == nil {
		ke.Spec.Registry.Override = map[string]string{}
	} // else return since overriding user from env might surprise me?
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], "IMAGE_") {
			name := strings.SplitN(pair[0], "_", 2)[1]
			ke.Spec.Registry.Override[name] = pair[1]
			// TODO: do we need this part for eventing as well?
			//switch name {
			//case "default":
			//	ke.Spec.Registry.Default = pair[1]
			//case "queue-proxy":
			//	Configure(ke, "deployment", "queueSidecarImage", pair[1])
			//	fallthrough
			//default:
			//	ke.Spec.Registry.Override[name] = pair[1]
			//}
		}
	}
	log.Info("Setting", "registry", ke.Spec.Registry)
	return nil
}

// TODO: delete!
// placeholder mutation
func logEventing(ke *eventingv1alpha1.KnativeEventing, c client.Client) error {
	Log.Info("Stage to mutate Eventing", ke)
	return nil
}
