package servicemesh

import (
	"context"
	"testing"
	"time"

	cetest "github.com/cloudevents/sdk-go/v2/test"
	"github.com/google/uuid"
	kafkafeatures "github.com/openshift-knative/serverless-operator/test/extensione2erekt/features"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"knative.dev/eventing-kafka-broker/control-plane/pkg/kafka"
	"knative.dev/eventing-kafka-broker/test/rekt/resources/configmap"
	duckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	"knative.dev/eventing/test/rekt/features"
	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/eventing/test/rekt/resources/containersource"
	"knative.dev/eventing/test/rekt/resources/subscription"
	"knative.dev/eventing/test/rekt/resources/trigger"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
	"knative.dev/reconciler-test/pkg/manifest"
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
				MatchEvent(cetest.HasType("dev.knative.eventing.samples.heartbeat")).
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

// ContainerSource -> KafkaChannel -> Subscription -> Sink (tenant-1)
//
//	-> subscription with reply to Sink (tenant-2)
//	-> subscription with deadLetterSink to Sink (tenant-1)
//
// The sink in "reply" should not receive any event because it's in a different tenant.
// The original sink and dead letter sink should receive an event.
func TestContainerSourceKafkaChannelKsvcWithReplyAndDLSCrossTenant(t *testing.T) {
	t.Parallel()

	// Do not use Knative Service Forwarder as it doesn't work correctly with replies.
	ctxTenant1, envTenant1 := global.Environment(
		environment.WithNamespace("tenant-1"),
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(5*time.Second, 4*time.Minute),
		environment.Managed(t),
	)
	ctxTenant2, envTenant2 := global.Environment(
		environment.WithNamespace("tenant-2"),
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(5*time.Second, 4*time.Minute),
		environment.Managed(t),
	)

	since := time.Now()
	sink := feature.MakeRandomK8sName("sink")

	// Deploy reply-sink in tenant-1.
	envTenant1.Test(ctxTenant1, t, deploySink(sink))
	// Check cross-tenant event.
	envTenant2.Test(ctxTenant2, t, verifyContainerSourceToChannelWithReplyAndDLS(sink, ctxTenant1, since))
}

func deploySink(sink string) *feature.Feature {
	f := feature.NewFeature()

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver))

	return f
}

func verifyContainerSourceToChannelWithReplyAndDLS(replySink string, replySinkCtx context.Context, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	channel := feature.MakeRandomK8sName("channel")
	sink := feature.MakeRandomK8sName("sink")
	dls := feature.MakeRandomK8sName("dls")
	cs := feature.MakeRandomK8sName("containersource")

	f.Setup("install channel", channel_impl.Install(channel))
	f.Setup("channel is ready", channel_impl.IsReady(channel))

	f.Setup("install sink", eventshub.Install(sink,
		eventshub.StartReceiver, eventshub.EchoEvent))
	f.Setup("install dls", eventshub.Install(dls, eventshub.StartReceiver))

	replySinkRef := service.AsKReference(replySink)
	replySinkRef.Namespace = environment.FromContext(replySinkCtx).Namespace()

	sub := feature.MakeRandomK8sName("subscription")
	f.Setup("install sink subscription", subscription.Install(sub,
		subscription.WithChannel(channel_impl.AsRef(channel)),
		subscription.WithSubscriber(service.AsKReference(sink), ""),
		subscription.WithReply(replySinkRef, ""),
		subscription.WithDeadLetterSink(service.AsKReference(dls), ""),
	))

	f.Setup("subscription is ready", subscription.IsReady(sub))

	f.Requirement("install containersource", containersource.Install(cs,
		containersource.WithSink(channel_impl.AsRef(channel), "")))
	f.Requirement("containersource goes ready", containersource.IsReady(cs))

	f.Assert("sink receives events", func(ctx context.Context, t feature.T) {
		assert.OnStore(sink).
			Match(features.HasKnNamespaceHeader(environment.FromContext(ctx).Namespace())).
			MatchEvent(cetest.HasType("dev.knative.eventing.samples.heartbeat")).
			AtLeast(1)(ctx, t)
	})

	f.Assert("deadlettersink receives events", func(ctx context.Context, t feature.T) {
		assert.OnStore(dls).
			Match(features.HasKnNamespaceHeader(environment.FromContext(ctx).Namespace())).
			MatchEvent(cetest.HasType("dev.knative.eventing.samples.heartbeat")).
			AtLeast(1)(ctx, t)
	})

	f.Assert("reply sink does not receive events", func(ctx context.Context, t feature.T) {
		assert.OnStore(replySink).
			Match(features.HasKnNamespaceHeader(environment.FromContext(ctx).Namespace())).
			MatchEvent(cetest.HasType("dev.knative.eventing.samples.heartbeat")).
			Not()(replySinkCtx, t)
	})

	f.Assert("request to event sink is forbidden", func(ctx context.Context, t feature.T) {
		kafkafeatures.VerifyRequestToSinkForbidden(replySink, environment.FromContext(replySinkCtx).Namespace(), since)
	})

	return f
}

// Source (Eventshub) -> KafkaBroker -> Trigger -> Ksvc -> Sink (Eventshub)
func TestSourceToKafkaBrokerKsvcCrossTenant(t *testing.T) {
	t.Parallel()

	ctxTenant1, envTenant1 := environmentWithNamespace(t, "tenant-1")
	ctxTenant2, envTenant2 := environmentWithNamespace(t, "tenant-2")

	broker := feature.MakeRandomK8sName("broker")
	sink := feature.MakeRandomK8sName("sink")

	// Deploy sink in tenant-1.
	envTenant1.Test(ctxTenant1, t, brokerTriggerKsvc(broker, sink))
	// Check cross-tenant event.
	envTenant2.Test(ctxTenant2, t, verifySourceToKafkaBrokerBlocked(broker, sink, ctxTenant1))
}

func brokerTriggerKsvc(brokerName, sink string) *feature.Feature {
	f := feature.NewFeatureNamed("broker smoke test")

	config := feature.MakeRandomK8sName("kafka-broker-config")
	triggerName := feature.MakeRandomK8sName("trigger")

	f.Setup("create broker config", configmap.Copy(
		types.NamespacedName{Namespace: system.Namespace(), Name: "kafka-broker-config"},
		config,
	))

	f.Setup("install sink", eventshub.Install(sink, eventshub.StartReceiver))

	f.Setup("install broker", broker.Install(brokerName,
		append([]manifest.CfgFn{broker.WithConfig(config)}, broker.WithBrokerClass(kafka.BrokerClass))...))
	f.Setup("broker ready", broker.IsReady(brokerName))

	backoffPolicy := duckv1.BackoffPolicyLinear
	f.Requirement("install trigger", trigger.Install(
		triggerName,
		brokerName,
		trigger.WithRetry(3, &backoffPolicy, pointer.String("PT1S")),
		trigger.WithSubscriber(service.AsKReference(sink), ""),
	))
	f.Requirement("trigger ready", trigger.IsReady(triggerName))

	return f
}

func verifySourceToKafkaBrokerBlocked(brokerName, sink string, sinkCtx context.Context) *feature.Feature {
	f := feature.NewFeature()

	event := cetest.FullEvent()
	event.SetID(uuid.New().String())

	eventMatchers := []cetest.EventMatcher{
		cetest.HasId(event.ID()),
		cetest.HasSource(event.Source()),
		cetest.HasType(event.Type()),
		cetest.HasSubject(event.Subject()),
	}

	f.Requirement("install eventshub source",
		func(ctx context.Context, t feature.T) {
			u, err := k8s.Address(sinkCtx, broker.GVR(), brokerName)
			if err != nil {
				t.Fatal(err)
			}
			eventshub.Install(
				feature.MakeRandomK8sName("source"),
				eventshub.StartSenderURL(u.String()),
				eventshub.InputEvent(event),
			)(ctx, t)
		},
	)

	f.Assert("source does not deliver event to kafka broker across tenants",
		func(ctx context.Context, t feature.T) {
			assert.OnStore(sink).
				MatchEvent(eventMatchers...).
				Not()(sinkCtx, t)
		},
	)

	// TODO: Assert istio-proxy logs automatically.
	//source pod:
	//{ "authority": "kafka-broker-ingress.knative-eventing.svc.cluster.local", "bytes_received": 7,
	//	"bytes_sent": 19, "downstream_local_address": "172.30.44.70:80", "downstream_peer_cert_v_end": "-",
	//	"downstream_peer_cert_v_start": "-", "downstream_remote_address": "10.129.3.19:36436",
	//	"downstream_tls_cipher": "-", "downstream_tls_version": "-", "duration": 2, "hostname": "source-wiyvyqvk",
	//	"istio_policy_status": "-", "method": "POST", "path": "/tenant-1/broker-mgwpiglw", "protocol": "HTTP/1.1", "request_duration": 0, "request_id": "27a61707-798c-400c-a3d5-49319dab0201", "requested_server_name": "-", "response_code": "403", "response_duration": 1, "response_tx_duration": 0, "response_flags": "-", "route_name": "default", "start_time": "2023-09-19T09:04:14.944Z", "upstream_cluster": "outbound|80||kafka-broker-ingress.knative-eventing.svc.cluster.local", "upstream_host": "10.128.4.89:8080", "upstream_local_address": "10.129.3.19:52354", "upstream_service_time": 1, "upstream_transport_failure_reason": "-", "user_agent": "Go-http-client/1.1", "x_forwarded_for": "-" }
	//kafka-broker-receiver:
	//{ "authority": "kafka-broker-ingress.knative-eventing.svc.cluster.local", "bytes_received": 0,
	//	"bytes_sent": 19, "downstream_local_address": "10.128.4.89:8080",
	//	"downstream_peer_cert_v_end": "2023-09-20T09:04:09.000Z", "downstream_peer_cert_v_start": "2023-09-19T09:02:09.000Z",
	//	"downstream_remote_address": "10.129.3.19:52354", "downstream_tls_cipher": "TLS_AES_256_GCM_SHA384",
	//	"downstream_tls_version": "TLSv1.3", "duration": 0, "hostname": "kafka-broker-receiver-566bbcd5c6-lxdc8",
	//	"istio_policy_status": "-", "method": "POST", "path": "/tenant-1/broker-mgwpiglw", "protocol": "HTTP/1.1",
	//	"request_duration": -, "request_id": "27a61707-798c-400c-a3d5-49319dab0201",
	//	"requested_server_name": "outbound_.80_._.kafka-broker-ingress.knative-eventing.svc.cluster.local",
	//	"response_code": "403", "response_duration": -, "response_tx_duration": -, "response_flags": "-",
	//	"route_name": "-", "start_time": "2023-09-19T09:04:14.947Z", "upstream_cluster": "inbound|8080||",
	//	"upstream_host": "-", "upstream_local_address": "-", "upstream_service_time": -,
	//	"upstream_transport_failure_reason": "-", "user_agent": "Go-http-client/1.1", "x_forwarded_for": "-" }

	return f
}
