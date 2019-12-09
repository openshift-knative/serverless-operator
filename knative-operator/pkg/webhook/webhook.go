package webhook

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var log = logf.Log.WithName("webhook")

// AddToManagerFuncs is a list of functions to add all Webhooks to the Manager
var AddToManagerFuncs []func(manager.Manager) (webhook.Webhook, error)

// AddToManager adds all Webhooks to the Manager
func AddToManager(m manager.Manager) error {

	if !runningOnOpenshift(m.GetConfig()) {
		log.Info("OpenShift not detected; no webhooks will be configured")
		return nil
	}

	webhooks := []webhook.Webhook{}
	for _, f := range AddToManagerFuncs {
		wh, err := f(m)
		if err != nil {
			log.Error(err, "Unable to setup webhook")
			return err
		}
		webhooks = append(webhooks, wh)
	}
	if len(webhooks) == 0 {
		return nil
	}

	log.Info("Setting up webhook server")
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// This will be started when the Manager is started
	as, err := webhook.NewServer("admission-webhook-server", m, webhook.ServerOptions{
		Port:    9876,
		CertDir: "/tmp/cert",
		BootstrapOptions: &webhook.BootstrapOptions{
			MutatingWebhookConfigName:   "mutating-knative-serving-openshift",
			ValidatingWebhookConfigName: "validating-knative-serving-openshift",
			Service: &webhook.Service{
				Namespace: operatorNs,
				Name:      "admission-server-service",
				// Selectors should select the pods that runs this webhook server.
				Selectors: map[string]string{
					"app": "openshift-admission-server",
				},
			},
		},
	})
	if err != nil {
		log.Error(err, "Unable to create a new webhook server")
		return err
	}

	log.Info("Registering webhooks to the webhook server")
	err = as.Register(webhooks...)
	if err != nil {
		log.Error(err, "Unable to register webhooks in the admission server")
		return err
	}
	return nil
}

func runningOnOpenshift(cfg *rest.Config) bool {
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		log.Error(err, "Can't create client")
		return false
	}
	gvk := schema.GroupVersionKind{Group: "route.openshift.io", Version: "v1", Kind: "route"}
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	if err := c.List(context.TODO(), nil, list); err != nil {
		if !meta.IsNoMatchError(err) {
			log.Error(err, "Unable to query for OpenShift Route")
		}
		return false
	}
	return true
}
