package eventinge2erekt

import (
	"context"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	"knative.dev/reconciler-test/pkg/feature"
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
	env.Test(ctx, t, VerifyPingSourceMetrics())

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, features.VerifyEncryptedTrafficToActivatorToApp(env.References(), since))
	}
}

func VerifyPingSourceMetrics() *feature.Feature {
	f := feature.NewFeature()

	f.Stable("pingsource").
		Must("has metrics", func(ctx context.Context, t feature.T) {
			if err := monitoringe2e.VerifyMetrics(ctx, monitoringe2e.EventingPingSourceMetricQueries); err != nil {
				t.Fatal("Failed to verify that PingSource data plane metrics work correctly", err)
			}
		})

	return f
}
