package servicemesh

import (
	"context"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/test"
	eventingfeatures "github.com/openshift-knative/serverless-operator/test/eventinge2erekt/features"
	"knative.dev/eventing/test/rekt/resources/pingsource"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/resources/service"
)

// PingSource (tenant-2) -> Ksvc (tenant-1) -> Sink (tenant-1)
func TestPingSourceToKsvcCrossTenant(t *testing.T) {
	t.Parallel()

	ctxTenant1, envTenant1 := environmentWithNamespace(t, "tenant-1")
	ctxTenant2, envTenant2 := environmentWithNamespace(t, "tenant-2")

	sink := feature.MakeRandomK8sName("sink")

	// Deploy sink in tenant-1.
	envTenant1.Test(ctxTenant1, t, DeployKsvcSink(sink))
	// Check cross-tenant event.
	envTenant2.Test(ctxTenant2, t, VerifyPingSourceToKsvcBlocked(ctxTenant1, sink, time.Now()))
}

func DeployKsvcSink(name string) *feature.Feature {
	f := feature.NewFeature()

	f.Setup("install sink", eventshub.Install(name, eventshub.StartReceiver))

	return f
}

func VerifyPingSourceToKsvcBlocked(sinkCtx context.Context, sink string, since time.Time) *feature.Feature {
	source := feature.MakeRandomK8sName("pingsource")
	f := feature.NewFeature()

	sinkRef := service.AsKReference(sink)
	sinkRef.Namespace = environment.FromContext(sinkCtx).Namespace()
	f.Requirement("install pingsource", pingsource.Install(source, pingsource.WithSink(sinkRef, "")))
	f.Requirement("pingsource goes ready", pingsource.IsReady(source))

	f.Assert("ping source does not deliver event to ksvc across tenants",
		func(ctx context.Context, t feature.T) {
			assert.OnStore(sink).
				MatchEvent(test.HasType("dev.knative.sources.ping")).
				Not()(sinkCtx, t)
		},
	)

	f.Assert("request to activator is forbidden", func(ctx context.Context, t feature.T) {
		eventingfeatures.VerifyEncryptedTrafficToActivator(since, true /*trafficBlocked*/)(sinkCtx, t)
	})

	return f
}
