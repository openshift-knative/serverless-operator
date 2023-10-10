package eventinge2erekt

import (
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/test/eventinge2erekt/features"
	"knative.dev/eventing/test/rekt/features/pingsource"
	"knative.dev/reconciler-test/pkg/environment"
)

// PingSource -> Ksvc -> Sink (Eventshub)
func TestPingSourceToKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	since := time.Now()

	env.Test(ctx, t, pingsource.SendsEventsWithSinkRef())

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, features.VerifyEncryptedTrafficToActivatorToApp(since))
	}
}
