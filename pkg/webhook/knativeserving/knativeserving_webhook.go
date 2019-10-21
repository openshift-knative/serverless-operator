package knativeserving

import (
	"context"
	"net/http"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

var log = logf.Log.WithName("webhook_knativeserving")

// Add creates a new KnativeServing Webhook
func Add(mgr manager.Manager) (webhook.Webhook, error) {
	log.Info("Setting up mutating webhook for KnativeServing")
	return builder.NewWebhookBuilder().
		Name("mutating.k8s.io").
		Mutating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(mgr).
		ForType(&servingv1alpha1.KnativeServing{}).
		Handlers(&KnativeServingConfigurator{}).
		Build()
}

// KnativeServingConfigurator annotates Kss
type KnativeServingConfigurator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
var _ admission.Handler = (*KnativeServingConfigurator)(nil)

// KnativeServingConfigurator adds an annotation to every incoming
// KnativeServing CR.
func (a *KnativeServingConfigurator) Handle(ctx context.Context, req types.Request) types.Response {
	ks := &servingv1alpha1.KnativeServing{}

	err := a.decoder.Decode(req, ks)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	copy := ks.DeepCopy()

	err = a.mutate(ctx, copy)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponse(ks, copy)
}

// mutate defaults the given ks
func (a *KnativeServingConfigurator) mutate(ctx context.Context, ks *servingv1alpha1.KnativeServing) error {
	stages := []func(context.Context, *servingv1alpha1.KnativeServing) error{
		a.egress,
		a.ingress,
	}
	for _, stage := range stages {
		if err := stage(ctx, ks); err != nil {
			return err
		}
	}
	log.Info("Webhook default stages complete")
	return nil
}

// configure egress
func (a *KnativeServingConfigurator) egress(ctx context.Context, ks *servingv1alpha1.KnativeServing) error {
	networkConfig := &configv1.Network{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: "cluster"}, networkConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return err
		}
		log.Info("No OpenShift cluster network config available")
		return nil
	}
	network := strings.Join(networkConfig.Spec.ServiceNetwork, ",")
	return configure(ks, "network", "istio.sidecar.includeOutboundIPRanges", network)
}

// configure ingress
func (a *KnativeServingConfigurator) ingress(ctx context.Context, ks *servingv1alpha1.KnativeServing) error {
	ingressConfig := &configv1.Ingress{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return err
		}
		log.Info("No OpenShift ingress config available")
		return nil
	}
	domain := ingressConfig.Spec.Domain
	if len(domain) > 0 {
		return configure(ks, "domain", domain, "")
	}
	return nil
}

// config helper to set value for key if not already set
func configure(ks *servingv1alpha1.KnativeServing, cm, key, value string) error {
	if ks.Spec.Config == nil {
		ks.Spec.Config = map[string]map[string]string{}
	}
	if len(ks.Spec.Config[cm][key]) == 0 {
		if ks.Spec.Config[cm] == nil {
			ks.Spec.Config[cm] = map[string]string{}
		}
		ks.Spec.Config[cm][key] = value
		log.Info("Configured", "map", cm, "key", key, "value", value)
	}
	return nil
}

// KnativeServingConfigurator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = (*KnativeServingConfigurator)(nil)

// InjectClient injects the client.
func (v *KnativeServingConfigurator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// KnativeServingConfigurator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = (*KnativeServingConfigurator)(nil)

// InjectDecoder injects the decoder.
func (v *KnativeServingConfigurator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}
