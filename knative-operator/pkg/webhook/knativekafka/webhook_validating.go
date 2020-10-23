package knativekafka

import (
	"context"
	"fmt"
	"net/http"
	"os"

	operatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Creates a new validating KnativeKafka Webhook
func ValidatingWebhook(mgr manager.Manager) (webhook.Webhook, error) {
	common.Log.Info("Setting up validating webhook for KnativeKafka")
	return builder.NewWebhookBuilder().
		Name("validating.knativekafka.openshift.io").
		Validating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(mgr).
		ForType(&operatorv1alpha1.KnativeKafka{}).
		Handlers(&KnativeKafkaValidator{}).
		Build()
}

// KnativeKafkaValidator validates KnativeKafka CR's
type KnativeKafkaValidator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*KnativeKafkaValidator)(nil)

// What makes us a webhook
func (v *KnativeKafkaValidator) Handle(ctx context.Context, req types.Request) types.Response {
	ke := &operatorv1alpha1.KnativeKafka{}

	err := v.decoder.Decode(req, ke)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validate(ctx, ke)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

// KnativeKafkaValidator checks for a minimum OpenShift version
func (v *KnativeKafkaValidator) validate(ctx context.Context, ke *operatorv1alpha1.KnativeKafka) (allowed bool, reason string, err error) {
	log := common.Log.WithName("validate")
	stages := []func(context.Context, *operatorv1alpha1.KnativeKafka) (bool, string, error){
		v.validateNamespace,
		v.validateLoneliness,
		v.validateShape,
		v.validateDependencies,
	}
	for _, stage := range stages {
		allowed, reason, err = stage(ctx, ke)
		if len(reason) > 0 {
			if err != nil {
				log.Error(err, reason)
			} else {
				log.Info(reason)
			}
		}
		if !allowed {
			return
		}
	}
	return
}

// KnativeKafkaValidator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = (*KnativeKafkaValidator)(nil)

// InjectClient injects the client.
func (v *KnativeKafkaValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// KnativeKafkaValidator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = (*KnativeKafkaValidator)(nil)

// InjectDecoder injects the decoder.
func (v *KnativeKafkaValidator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}

// validate required namespace, if any
func (v *KnativeKafkaValidator) validateNamespace(ctx context.Context, ke *operatorv1alpha1.KnativeKafka) (bool, string, error) {
	ns, required := os.LookupEnv("REQUIRED_KAFKA_NAMESPACE")
	if required && ns != ke.Namespace {
		return false, fmt.Sprintf("KnativeKafka may only be created in %s namespace", ns), nil
	}
	return true, "", nil
}

// validate this is the only KE in this namespace
func (v *KnativeKafkaValidator) validateLoneliness(ctx context.Context, ke *operatorv1alpha1.KnativeKafka) (bool, string, error) {
	list := &operatorv1alpha1.KnativeKafkaList{}
	if err := v.client.List(ctx, &client.ListOptions{Namespace: ke.Namespace}, list); err != nil {
		return false, "Unable to list KnativeKafkas", err
	}
	for _, v := range list.Items {
		if ke.Name != v.Name {
			return false, "Only one KnativeKafka allowed per namespace", nil
		}
	}
	return true, "", nil
}

// validate the shape of the CR
func (v *KnativeKafkaValidator) validateShape(_ context.Context, ke *operatorv1alpha1.KnativeKafka) (bool, string, error) {
	if ke.Spec.Channel.Enabled && ke.Spec.Channel.BootstrapServers == "" {
		return false, "spec.channel.bootStrapServers is a required detail when spec.channel.enabled is true", nil
	}
	return true, "", nil
}

// validate that KnativeEventing is installed as a hard dep
func (v *KnativeKafkaValidator) validateDependencies(ctx context.Context, ke *operatorv1alpha1.KnativeKafka) (bool, string, error) {
	// check to see if we can find KnativeEventing
	list := &eventingv1alpha1.KnativeEventingList{}
	if err := v.client.List(ctx, &client.ListOptions{Namespace: ke.Namespace}, list); err != nil {
		return false, "Unable to list KnativeEventing instance", err
	}
	if len(list.Items) == 0 {
		return false, "KnativeEventing instance must be installed before KnativeKafka", nil
	}
	// successful case
	return true, "", nil
}
