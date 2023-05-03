package eventinge2e_rekt

import (
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/test/eventinge2e_rekt/features"
	"knative.dev/eventing/test/rekt/features/pingsource"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

func TestPingSourceWithSinkRef(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		// Enables KnativeService in the PingSource scenario.
		eventshub.WithKnativeServiceForwarder,
		environment.Managed(t),
	)

	since := time.Now()

	env.Test(ctx, t, pingsource.SendsEventsWithSinkRef())

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, features.VerifyEncryptedTrafficToActivatorToApp(env.References(), since))
	}
}
