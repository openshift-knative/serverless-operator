package extensione2erekt

import (
	"testing"
	"time"

	eventingfeatures "github.com/openshift-knative/serverless-operator/test/eventinge2erekt/features"
	kafkafeatures "github.com/openshift-knative/serverless-operator/test/extensione2erekt/features"
	"knative.dev/eventing-kafka-broker/test/rekt/features"
	"knative.dev/reconciler-test/pkg/environment"
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

	since := time.Now()

	env.Test(ctx, t, features.KafkaSourceTLS(kafkaSource, kafkaSink, topic))

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, eventingfeatures.VerifyEncryptedTrafficToActivatorToApp(env.References(), since))
		env.Test(ctx, t, kafkafeatures.VerifyEncryptedTrafficToKafkaSink(kafkaSink, since))
	}
}

func TestKafkaSourceSASL(t *testing.T) {

	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, features.KafkaSourceSASL())
}
