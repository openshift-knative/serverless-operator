package extensione2erekt

import (
	"testing"

	"knative.dev/eventing-kafka-broker/test/rekt/features"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

func TestKafkaSourceBinaryEvent(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	env.Test(ctx, t, features.KafkaSourceBinaryEvent())
}

func TestKafkaSourceTLS(t *testing.T) {

	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	kafkaSource := feature.MakeRandomK8sName("kafkaSource")
	kafkaSink := feature.MakeRandomK8sName("kafkaSink")
	topic := feature.MakeRandomK8sName("topic")

	env.Test(ctx, t, features.KafkaSourceTLS(kafkaSource, kafkaSink, topic))
}

func TestKafkaSourceSASL(t *testing.T) {

	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	env.Test(ctx, t, features.KafkaSourceSASL())
}
