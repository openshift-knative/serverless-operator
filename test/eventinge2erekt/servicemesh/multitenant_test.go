package servicemesh

import (
	"context"
	"testing"

	"github.com/cloudevents/sdk-go/v2/test"
	"knative.dev/eventing/test/rekt/resources/pingsource"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/resources/service"
)

// PingSource -> Ksvc -> Sink (Eventshub)
func TestPingSourceToKsvcCrossTenant(t *testing.T) {
	t.Parallel()

	ctxTenant1, envTenant1 := environmentWithNamespace(t, "tenant-1")
	ctxTenant2, envTenant2 := environmentWithNamespace(t, "tenant-2")

	sink := feature.MakeRandomK8sName("sink")

	// Deploy sink in tenant-1.
	envTenant1.Test(ctxTenant1, t, ksvcSink(sink))
	// Check cross-tenant event.
	envTenant2.Test(ctxTenant2, t, verifyPingSourceToKsvcBlocked(sink, ctxTenant1))
}

func ksvcSink(name string) *feature.Feature {
	f := feature.NewFeature()

	f.Setup("install sink", eventshub.Install(name, eventshub.StartReceiver))

	return f
}

func verifyPingSourceToKsvcBlocked(sink string, sinkCtx context.Context) *feature.Feature {
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

	// TODO: The Activator Pod's istio-proxy throws 403 as expected:
	// { "authority": "sink-xshqhaao.tenant-1.svc.cluster.local",
	//"bytes_received": 0, "bytes_sent": 19, "downstream_local_address": "10.128.4.21:8012",
	//"downstream_peer_cert_v_end": "2023-09-19T09:23:12.000Z", "downstream_peer_cert_v_start":
	//"2023-09-18T09:21:12.000Z", "downstream_remote_address": "10.131.3.179:54072",
	//"downstream_tls_cipher": "TLS_AES_256_GCM_SHA384", "downstream_tls_version": "TLSv1.3",
	//"duration": 0, "hostname": "activator-69b7f975bb-5rhsx", "istio_policy_status": "-",
	//"method": "POST", "path": "/", "protocol": "HTTP/1.1", "request_duration": 0, "request_id":
	//"a00c9d1d-daa7-432c-8b98-196c3dbf4dee", "requested_server_name":
	//"outbound_.80_._.sink-xshqhaao-00001.tenant-1.svc.cluster.local",
	//"response_code": "403", "response_duration": -, "response_tx_duration": -,
	//"response_flags": "-", "route_name": "-", "start_time": "2023-09-18T11:11:00.370Z",

	return f
}
