package servicemesh

import (
	"context"
	"testing"

	"github.com/cloudevents/sdk-go/v2/test"
	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/eventing/test/rekt/resources/containersource"
	"knative.dev/eventing/test/rekt/resources/subscription"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/resources/service"
)

// ContainerSource -> KafkaChannel -> Subscription -> Ksvc -> Sink (Eventshub)
func TestContainerSourceKafkaChannelKsvcCrossTenant(t *testing.T) {
	t.Parallel()

	ctxTenant1, envTenant1 := environmentWithNamespace(t, "tenant-1")
	ctxTenant2, envTenant2 := environmentWithNamespace(t, "tenant-2")

	channel := feature.MakeRandomK8sName("channel")
	sink := feature.MakeRandomK8sName("sink")

	// Deploy sink in tenant-1.
	envTenant1.Test(ctxTenant1, t, kafkaChannelKsvc(channel, sink))
	// Check cross-tenant event.
	envTenant2.Test(ctxTenant2, t, verifyContainerSourceToChannelBlocked(channel, sink, ctxTenant1))
}

func kafkaChannelKsvc(channel, sink string) *feature.Feature {
	f := feature.NewFeature()

	f.Setup("install channel", channel_impl.Install(channel))
	f.Setup("channel is ready", channel_impl.IsReady(channel))

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver))

	sub := feature.MakeRandomK8sName("subscription")
	f.Setup("install sink subscription", subscription.Install(sub,
		subscription.WithChannel(channel_impl.AsRef(channel)),
		subscription.WithSubscriber(service.AsKReference(sink), ""),
	))

	f.Setup("subscription is ready", subscription.IsReady(sub))

	return f
}

func verifyContainerSourceToChannelBlocked(channel, sink string, channelCtx context.Context) *feature.Feature {
	f := feature.NewFeature()

	cs := feature.MakeRandomK8sName("containersource")

	channelRef := channel_impl.AsRef(channel)
	channelRef.Namespace = environment.FromContext(channelCtx).Namespace()

	f.Setup("install containersource", containersource.Install(cs, containersource.WithSink(channelRef, "")))
	f.Setup("containersource goes ready", containersource.IsReady(cs))

	f.Assert("container source does not deliver event to channel across tenants",
		func(ctx context.Context, t feature.T) {
			assert.OnStore(sink).
				MatchEvent(test.HasType("dev.knative.eventing.samples.heartbeat")).
				Not()(channelCtx, t)
		},
	)

	// TODO: Verify istio-proxy logs automatically:
	//container source pod:
	//{ "authority": "channel-rxjpvaev-kn-channel.tenant-1.svc.cluster.local", "bytes_received": 19, "bytes_sent": 19,
	//	"downstream_local_address": "172.30.8.184:80", "downstream_peer_cert_v_end": "-",
	//	"downstream_peer_cert_v_start": "-", "downstream_remote_address": "10.128.4.87:57748", "downstream_tls_cipher": "-",
	//	"downstream_tls_version": "-", "duration": 2, "hostname": "containersource-gmjuflkm-deployment-85c8c487b6-tjlxg",
	//	"istio_policy_status": "-", "method": "POST", "path": "/", "protocol": "HTTP/1.1", "request_duration": 0, "request_id":
	//	"dc70c751-57b3-4989-ad16-2f11c9841099", "requested_server_name": "-", "response_code": "403", "response_duration": 2,
	//	"response_tx_duration": 0, "response_flags": "-", "route_name": "default", "start_time": "2023-09-19T07:42:07.245Z",
	//	"upstream_cluster": "outbound|80||channel-rxjpvaev-kn-channel.tenant-1.svc.cluster.local", "upstream_host":
	//	"172.30.8.184:80", "upstream_local_address": "10.128.4.87:57752", "upstream_service_time": 2,
	//	"upstream_transport_failure_reason": "...

	//kafka-channel-receiver:
	//{ "authority": "channel-rxjpvaev-kn-channel.tenant-1.svc.cluster.local",
	//	"bytes_received": 0, "bytes_sent": 19, "downstream_local_address": "10.128.2.232:8080",
	//	"downstream_peer_cert_v_end": "2023-09-20T07:41:51.000Z", "downstream_peer_cert_v_start":
	//	"2023-09-19T07:39:51.000Z", "downstream_remote_address": "10.128.4.87:36012", "downstream_tls_cipher":
	//	"TLS_AES_256_GCM_SHA384", "downstream_tls_version": "TLSv1.3", "duration": 0, "hostname":
	//	"kafka-channel-receiver-5d8f99847b-fjtmj", "istio_policy_status": "-",
	//	"method": "POST", "path": "/", "protocol": "HTTP/1.1", "request_duration": -,
	//	"request_id": "2cb01579-cc17-4600-b880-ef9b5faadc72", "requested_server_name":
	//	"outbound_.80_._.channel-rxjpvaev-kn-channel.tenant-1.svc.cluster.local", "response_code": "403",
	//	"response_duration": -, "response_tx_duration": -, "response_flags": "-", "route_name": "-",
	//	"start_time": "2023-09-19T07:43:17.249Z", "upstream_cluster": "inbound|8080||", "upstream_host": "-",
	//	"upstream_local_address": "-", "upstream_service_time": -, "upstream_transport_failure_reason": "-",
	//	"user_agent": "Go-http-client/1.1", "x_forwarded_for": "-" }

	return f
}
