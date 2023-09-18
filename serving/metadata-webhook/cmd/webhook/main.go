package main

import (
	"context"

	"github.com/openshift-knative/serverless-operator/serving/metadata-webhook/pkg/defaults"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/certificates"
	"knative.dev/pkg/webhook/resourcesemantics"
	"knative.dev/pkg/webhook/resourcesemantics/defaulting"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1beta1 "knative.dev/serving/pkg/apis/serving/v1beta1"
)

var types = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
	servingv1.SchemeGroupVersion.WithKind("Service"):           &defaults.TargetKService{},
	servingv1.SchemeGroupVersion.WithKind("Route"):             &defaults.TargetRoute{},
	servingv1.SchemeGroupVersion.WithKind("Configurtion"):      &defaults.TargetConfiguration{},
	servingv1beta1.SchemeGroupVersion.WithKind("DomainMappig"): &defaults.TargetDomainMapping{},
}

func NewDefaultingAdmissionController(ctx context.Context, _ configmap.Watcher) *controller.Impl {
	return defaulting.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"webhook.metadata-webhook.example.com",

		// The path on which to serve the webhook.
		"/defaulting",

		// The resources to default.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			// Here is where you would infuse the context with state
			// (e.g. attach a store with configmap data)
			return ctx
		},

		// Whether to disallow unknown fields.
		true,
	)
}

func main() {
	ctx := webhook.WithOptions(signals.NewContext(), webhook.Options{
		ServiceName: "webhook",
		Port:        8443,
		SecretName:  "webhook-certs",
	})

	sharedmain.WebhookMainWithContext(ctx, "webhook",
		certificates.NewController,
		NewDefaultingAdmissionController,
	)
}
