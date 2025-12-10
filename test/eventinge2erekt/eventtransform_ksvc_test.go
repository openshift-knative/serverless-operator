package eventinge2erekt

import (
	"testing"

	"knative.dev/eventing/test/rekt/features/eventtransform"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

func TestEventTransform(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		// EventTransform does not work with ServiceMesh
		t.Skip("https://issues.redhat.com/browse/SRVKE-1751")
	}

	env.Test(ctx, t, eventtransform.JsonataDirect())
	env.Test(ctx, t, eventtransform.JsonataSink())
	env.Test(ctx, t, eventtransform.JsonataSinkReplyTransform())
}
