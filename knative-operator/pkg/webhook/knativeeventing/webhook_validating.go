package knativeeventing

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Validator validates KnativeEventing CR's
type Validator struct {
	client  client.Client
	decoder admission.Decoder
}

// NewValidator creates a new Valicator instance to validate KnativeEventing CRs.
func NewValidator(client client.Client, decoder admission.Decoder) *Validator {
	return &Validator{
		client:  client,
		decoder: decoder,
	}
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*Validator)(nil)

// Handle implements the Handler interface.
func (v *Validator) Handle(ctx context.Context, req admission.Request) admission.Response {
	ke := &operatorv1beta1.KnativeEventing{}

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
func (v *Validator) validate(ctx context.Context, ke *operatorv1beta1.KnativeEventing) (allowed bool, reason string, err error) {
	log := common.Log.WithName("validate")
	stages := []func(context.Context, *operatorv1beta1.KnativeEventing) (bool, string, error){
		v.validateNamespace,
		v.validateLoneliness,
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

// validate required namespace, if any
func (v *Validator) validateNamespace(_ context.Context, ke *operatorv1beta1.KnativeEventing) (bool, string, error) {
	ns, required := os.LookupEnv("REQUIRED_EVENTING_NAMESPACE")
	if required && ns != ke.Namespace {
		return false, fmt.Sprintf("KnativeEventing may only be created in %s namespace", ns), nil
	}
	return true, "", nil
}

// validate this is the only KE in this namespace
func (v *Validator) validateLoneliness(ctx context.Context, ke *operatorv1beta1.KnativeEventing) (bool, string, error) {
	list := &operatorv1beta1.KnativeEventingList{}
	if err := v.client.List(ctx, list, &client.ListOptions{Namespace: ke.Namespace}); err != nil {
		return false, "Unable to list KnativeEventings", err
	}
	for _, v := range list.Items {
		if ke.Name != v.Name {
			return false, "Only one KnativeEventing allowed per namespace", nil
		}
	}
	return true, "", nil
}
