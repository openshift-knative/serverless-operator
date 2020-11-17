package knativekafka

import (
	"context"
	"fmt"
	"net/http"
	"os"

	operatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Validator validates KnativeKafka CR's
type Validator struct {
	client  client.Client
	decoder *admission.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*Validator)(nil)

// What makes us a webhook
func (v *Validator) Handle(ctx context.Context, req admission.Request) admission.Response {
	ke := &operatorv1alpha1.KnativeKafka{}

	err := v.decoder.Decode(req, ke)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validate(ctx, ke)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

// Validator checks for a minimum OpenShift version
func (v *Validator) validate(ctx context.Context, ke *operatorv1alpha1.KnativeKafka) (allowed bool, reason string, err error) {
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

// Validator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = (*Validator)(nil)

// InjectClient injects the client.
func (v *Validator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// Validator implements inject.Decoder.
// A decoder will be automatically injected.
var _ admission.DecoderInjector = (*Validator)(nil)

// InjectDecoder injects the decoder.
func (v *Validator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// validate required namespace, if any
func (v *Validator) validateNamespace(ctx context.Context, ke *operatorv1alpha1.KnativeKafka) (bool, string, error) {
	ns, required := os.LookupEnv("REQUIRED_KAFKA_NAMESPACE")
	if required && ns != ke.Namespace {
		return false, fmt.Sprintf("KnativeKafka may only be created in %s namespace", ns), nil
	}
	return true, "", nil
}

// validate this is the only KE in this namespace
func (v *Validator) validateLoneliness(ctx context.Context, ke *operatorv1alpha1.KnativeKafka) (bool, string, error) {
	list := &operatorv1alpha1.KnativeKafkaList{}
	if err := v.client.List(ctx, list, &client.ListOptions{Namespace: ke.Namespace}); err != nil {
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
func (v *Validator) validateShape(_ context.Context, ke *operatorv1alpha1.KnativeKafka) (bool, string, error) {
	if ke.Spec.Channel.Enabled && ke.Spec.Channel.BootstrapServers == "" {
		return false, "spec.channel.bootStrapServers is a required detail when spec.channel.enabled is true", nil
	}
	if ke.Spec.Channel.AuthSecretName != "" && ke.Spec.Channel.AuthSecretNamespace == "" {
		return false, "spec.channel.authSecretNamespace is required when spec.channel.authSecretName is defined", nil
	}
	if ke.Spec.Channel.AuthSecretNamespace != "" && ke.Spec.Channel.AuthSecretName == "" {
		return false, "spec.channel.authSecretName is required when spec.channel.authSecretNamespace is defined", nil
	}
	return true, "", nil
}

// validate that KnativeEventing is installed as a hard dep
func (v *Validator) validateDependencies(ctx context.Context, ke *operatorv1alpha1.KnativeKafka) (bool, string, error) {
	// check to see if we can find KnativeEventing
	list := &eventingv1alpha1.KnativeEventingList{}
	if err := v.client.List(ctx, list, &client.ListOptions{Namespace: ke.Namespace}); err != nil {
		return false, "Unable to list KnativeEventing instance", err
	}
	if len(list.Items) == 0 {
		return false, "KnativeEventing instance must be installed before KnativeKafka", nil
	}
	// successful case
	return true, "", nil
}
