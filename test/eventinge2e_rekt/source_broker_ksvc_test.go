package eventinge2e_rekt

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test/eventinge2e_rekt/features"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

func TestPingSourceBrokerTriggerKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithConfigOptions("eventshub", eventshub.WithKnativeServiceForwarder),
		environment.Managed(t),
	)

	env.Test(ctx, t, features.SendsEventsWithSinkRef())
}
