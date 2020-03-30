// +build eventing

package webhook

import (
	ke "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeeventing"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, ke.MutatingWebhook, ke.ValidatingWebhook)
}
