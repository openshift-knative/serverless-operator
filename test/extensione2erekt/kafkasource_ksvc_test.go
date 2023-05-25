package extensione2erekt

import (
	"testing"

	"knative.dev/eventing-kafka-broker/test/rekt/features"
	"knative.dev/reconciler-test/pkg/feature"
)

func TestKafkaSourceBinaryEvent(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, features.KafkaSourceBinaryEvent())
}

func TestKafkaSourceTLS(t *testing.T) {

	t.Parallel()

	ctx, env := defaultEnvironment(t)

	kafkaSource := feature.MakeRandomK8sName("kafkaSource")
	kafkaSink := feature.MakeRandomK8sName("kafkaSink")
	topic := feature.MakeRandomK8sName("topic")

	env.Test(ctx, t, features.KafkaSourceTLS(kafkaSource, kafkaSink, topic))
}

func TestKafkaSourceSASL(t *testing.T) {

	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, features.KafkaSourceSASL())
}
