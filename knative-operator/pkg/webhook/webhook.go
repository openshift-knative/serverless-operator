package webhook

import (
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var log = logf.Log.WithName("webhook")

// AddToManagerFuncs is a list of functions to add all Webhooks to the Manager
var AddToManagerFuncs []func(manager.Manager) (webhook.Webhook, error)

// AddToManager adds all Webhooks to the Manager
func AddToManager(m manager.Manager) error {

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
			MutatingWebhookConfigName:   "mutating-knative-openshift",
			ValidatingWebhookConfigName: "validating-knative-openshift",
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
