package knativeserving

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Validator validates KnativeServing CR's
type Validator struct {
	Client  client.Client
	Decoder *admission.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*Validator)(nil)

// What makes us a webhook
func (v *Validator) Handle(ctx context.Context, req admission.Request) admission.Response {
	ks := &servingv1alpha1.KnativeServing{}

	err := v.Decoder.Decode(req, ks)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validate(ctx, ks)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

// Validator checks for a minimum OpenShift version
func (v *Validator) validate(ctx context.Context, ks *servingv1alpha1.KnativeServing) (allowed bool, reason string, err error) {
	log := common.Log.WithName("validate")
	stages := []func(context.Context, *servingv1alpha1.KnativeServing) (bool, string, error){
		v.validateNamespace,
		v.validateLoneliness,
	}
	for _, stage := range stages {
		allowed, reason, err = stage(ctx, ks)
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
func (v *Validator) validateNamespace(ctx context.Context, ks *servingv1alpha1.KnativeServing) (bool, string, error) {
	ns, required := os.LookupEnv("REQUIRED_SERVING_NAMESPACE")
	if required && ns != ks.Namespace {
		return false, fmt.Sprintf("KnativeServing may only be created in %s namespace", ns), nil
	}
	return true, "", nil
}

// validate this is the only KS in this namespace
func (v *Validator) validateLoneliness(ctx context.Context, ks *servingv1alpha1.KnativeServing) (bool, string, error) {
	list := &servingv1alpha1.KnativeServingList{}
	if err := v.Client.List(ctx, list, &client.ListOptions{Namespace: ks.Namespace}); err != nil {
		return false, "Unable to list KnativeServings", err
	}
	for _, v := range list.Items {
		if ks.Name != v.Name {
			return false, "Only one KnativeServing allowed per namespace", nil
		}
	}
	return true, "", nil
}
