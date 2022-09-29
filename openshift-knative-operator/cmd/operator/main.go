package main

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/operator/pkg/apis/operator"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/knativeeventing"
	"knative.dev/operator/pkg/reconciler/knativeserving"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/certificates"
	"knative.dev/pkg/webhook/resourcesemantics/conversion"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/eventing"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/serving"
)

func main() {
	// Set up a signal context with our webhook options
	ctx := webhook.WithOptions(signals.NewContext(), webhook.Options{
		ServiceName: webhook.NameFromEnv(),
		Port:        webhook.PortFromEnv(8443),
		SecretName:  "operator-webhook-certs",
	})

	sharedmain.WebhookMainWithContext(ctx, "knative-operator",
		certificates.NewController,
		newConversionController,
		knativeeventing.NewExtendedController(eventing.NewExtension),
		knativeserving.NewExtendedController(serving.NewExtension),
	)
}

func newConversionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	var (
		v1beta1  = operatorv1beta1.SchemeGroupVersion.Version
		v1alpha1 = operatorv1alpha1.SchemeGroupVersion.Version
	)

	return conversion.NewConversionController(ctx,
		// The path on which to serve the webhook
		"/resource-conversion",

		// Specify the types of custom resource definitions that should be converted
		map[schema.GroupKind]conversion.GroupKindConversion{
			operatorv1beta1.Kind("KnativeServing"): {
				DefinitionName: operator.KnativeServingResource.String(),
				HubVersion:     v1alpha1,
				Zygotes: map[string]conversion.ConvertibleObject{
					v1alpha1: &operatorv1alpha1.KnativeServing{},
					v1beta1:  &operatorv1beta1.KnativeServing{},
				},
			},
			operatorv1beta1.Kind("KnativeEventing"): {
				DefinitionName: operator.KnativeEventingResource.String(),
				HubVersion:     v1alpha1,
				Zygotes: map[string]conversion.ConvertibleObject{
					v1alpha1: &operatorv1alpha1.KnativeEventing{},
					v1beta1:  &operatorv1beta1.KnativeEventing{},
				},
			},
		},

		// A function that infuses the context passed to ConvertTo/ConvertFrom/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return ctx
		},
	)
}
