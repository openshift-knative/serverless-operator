package features

import (
	"context"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/brokerconfig"
	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/inmemorychannel"
	ksvcresources "github.com/openshift-knative/serverless-operator/test/kitchensinke2e/ksvc"
	testpkg "knative.dev/eventing-kafka-broker/test/pkg"
	kafkachannelresources "knative.dev/eventing-kafka-broker/test/rekt/resources/kafkachannel"
	"knative.dev/eventing-kafka-broker/test/rekt/resources/kafkasink"
	"knative.dev/eventing-kafka-broker/test/rekt/resources/kafkatopic"
	"knative.dev/eventing/test/rekt/resources/apiserversource"
	brokerresources "knative.dev/eventing/test/rekt/resources/broker"
	channelresources "knative.dev/eventing/test/rekt/resources/channel"
	"knative.dev/eventing/test/rekt/resources/containersource"
	parallelresources "knative.dev/eventing/test/rekt/resources/parallel"
	"knative.dev/eventing/test/rekt/resources/pingsource"
	sequenceresources "knative.dev/eventing/test/rekt/resources/sequence"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/manifest"
	svcresources "knative.dev/reconciler-test/resources/svc"
)

const (
	NumDeployments = 3
)

/*
Components are used as bogus sinks/filters/replies. We only test that the whole system becomes Ready,
so we don't implement any kind of dataplane in the components.
*/

func withKafkaChannelTemplate() manifest.CfgFn {
	return func(cfg map[string]interface{}) {
		cfg["channelTemplate"] = map[string]interface{}{
			"apiVersion": kafkachannelresources.GVR().GroupVersion().String(),
			"kind":       "KafkaChannel",
			"spec": map[string]string{
				"replicationFactor": "3",
				"numPartitions":     "10",
			},
		}
	}
}

func withInMemoryChannelTemplate() manifest.CfgFn {
	return func(cfg map[string]interface{}) {
		cfg["channelTemplate"] = map[string]interface{}{
			"apiVersion": inmemorychannel.GVR().GroupVersion().String(),
			"kind":       "InMemoryChannel",
		}
	}
}

var kafkaChannel = genericComponent{
	shortLabel: "kc",
	label:      "KafkaChannel",
	kind:       "KafkaChannel",
	gvr:        kafkachannelresources.GVR(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		defaultOpts := []manifest.CfgFn{
			kafkachannelresources.WithNumPartitions("10"),
			kafkachannelresources.WithReplicationFactor("3"),
		}

		opts = append(defaultOpts, opts...)

		return kafkachannelresources.Install(name,
			opts...,
		)
	},
}

var inMemoryChannel = genericComponent{
	shortLabel: "imc",
	label:      "InMemoryChannel",
	kind:       "InMemoryChannel",
	gvr:        inmemorychannel.GVR(),
	install:    inmemorychannel.Install,
}

var genericChannelWithKafkaChannelTemplate = genericComponent{
	shortLabel: "gkc",
	label:      "Channel(KafkaChannel)",
	kind:       "Channel",
	gvr:        channelresources.GVR(),
	install: func(name string, _ ...manifest.CfgFn) feature.StepFn {
		return channelresources.Install(name, withKafkaChannelTemplate())
	},
}

var genericChannelWithInMemoryChannelTemplate = genericComponent{
	shortLabel: "gimc",
	label:      "Channel(InMemoryChannel)",
	kind:       "Channel",
	gvr:        channelresources.GVR(),
	install: func(name string, _ ...manifest.CfgFn) feature.StepFn {
		return channelresources.Install(name, withInMemoryChannelTemplate())
	},
}

var ksvc = genericComponent{
	shortLabel: "ksvc",
	label:      "Knative Service",
	kind:       "Service",
	gvr:        ksvcresources.GVR(),
	install: func(name string, _ ...manifest.CfgFn) feature.StepFn {
		return ksvcresources.Install(name)
	},
}

var inMemoryChannelMtBroker = genericComponent{
	shortLabel: "imcmtb",
	label:      "MTBroker(InMemoryChannel)",
	kind:       "Broker",
	gvr:        brokerresources.GVR(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			brokerconfig.Install(name, brokerconfig.WithInMemoryChannelMTBroker())(ctx, t)

			brokerresources.Install(name,
				append([]manifest.CfgFn{
					brokerresources.WithBrokerClass("MTChannelBasedBroker"),
					brokerresources.WithConfig(name)},
					opts...)...)(ctx, t)
		}
	},
}

var kafkaChannelMtBroker = genericComponent{
	shortLabel: "kcmtb",
	label:      "MTBroker(KafkaChannel)",
	kind:       "Broker",
	gvr:        brokerresources.GVR(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			brokerconfig.Install(name, brokerconfig.WithKafkaChannelMTBroker())(ctx, t)

			brokerresources.Install(name,
				append([]manifest.CfgFn{
					brokerresources.WithBrokerClass("MTChannelBasedBroker"),
					brokerresources.WithConfig(name)},
					opts...)...)(ctx, t)
		}
	},
}

var kafkaBroker = genericComponent{
	shortLabel: "kb",
	label:      "KafkaBroker",
	kind:       "Broker",
	gvr:        brokerresources.GVR(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			brokerconfig.Install(name, brokerconfig.WithKafkaBroker())(ctx, t)

			brokerresources.Install(name,
				append([]manifest.CfgFn{
					brokerresources.WithBrokerClass("Kafka"),
					brokerresources.WithConfig(name)},
					opts...)...)(ctx, t)
		}
	},
}

var inMemoryChannelSequence = genericComponent{
	shortLabel: "imcseq",
	label:      "Sequence(InMemoryChannel)",
	kind:       "Sequence",
	gvr:        sequenceresources.GVR(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			// We'll populate the sequence with two bogus services, just so it's not empty
			step1 := name + "-s1"
			step2 := name + "-s2"
			svcresources.Install(step1, "app", step1)(ctx, t)
			svcresources.Install(step2, "app", step2)(ctx, t)

			sequenceresources.Install(name,
				withInMemoryChannelTemplate(),
				sequenceresources.WithStep(svcresources.AsKReference(step1), ""),
				sequenceresources.WithStep(svcresources.AsKReference(step2), ""),
			)(ctx, t)
		}
	},
}

var kafkaChannelSequence = genericComponent{
	shortLabel: "kcseq",
	label:      "Sequence(KafkaChannel)",
	kind:       "Sequence",
	gvr:        sequenceresources.GVR(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			step1 := name + "-s1"
			step2 := name + "-s2"
			svcresources.Install(step1, "app", step1)(ctx, t)
			svcresources.Install(step2, "app", step2)(ctx, t)

			sequenceresources.Install(name,
				withKafkaChannelTemplate(),
				sequenceresources.WithStep(svcresources.AsKReference(step1), ""),
				sequenceresources.WithStep(svcresources.AsKReference(step2), ""),
			)(ctx, t)
		}
	},
}

var inMemoryChannelParallel = genericComponent{
	shortLabel: "imcpar",
	label:      "Parallel(InMemoryChannel)",
	kind:       "Parallel",
	gvr:        parallelresources.GVR(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			branch1 := name + "-b1"
			branch2 := name + "-b2"
			reply := name + "-r"

			svcresources.Install(branch1, "app", branch1)(ctx, t)
			svcresources.Install(branch2, "app", branch2)(ctx, t)
			inMemoryChannel.Install(reply)(ctx, t)

			parallelresources.Install(name,
				withInMemoryChannelTemplate(),
				parallelresources.WithSubscriberAt(0, svcresources.AsKReference(branch1), ""),
				parallelresources.WithSubscriberAt(1, svcresources.AsKReference(branch2), ""),
				parallelresources.WithReply(inMemoryChannel.KReference(reply), ""),
			)(ctx, t)
		}
	},
}

var kafkaChannelParallel = genericComponent{
	shortLabel: "kcpar",
	label:      "Parallel(KafkaChannel)",
	kind:       "Parallel",
	gvr:        parallelresources.GVR(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			branch1 := name + "-b1"
			branch2 := name + "-b2"
			reply := name + "-r"

			svcresources.Install(branch1, "app", branch1)(ctx, t)
			svcresources.Install(branch2, "app", branch2)(ctx, t)
			kafkaChannel.Install(reply)(ctx, t)

			parallelresources.Install(name,
				withKafkaChannelTemplate(),
				parallelresources.WithSubscriberAt(0, svcresources.AsKReference(branch1), ""),
				parallelresources.WithSubscriberAt(1, svcresources.AsKReference(branch2), ""),
				parallelresources.WithReply(kafkaChannel.KReference(reply), ""),
			)(ctx, t)
		}
	},
}

var kafkaSink = genericComponent{
	shortLabel: "kasi",
	label:      "KafkaSink",
	kind:       "KafkaSink",
	gvr:        kafkasink.GVR(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			topic := name + "-t"
			kafkatopic.Install(topic)
			kafkasink.Install(name, topic, testpkg.BootstrapServersPlaintextArr,
				kafkasink.WithNumPartitions(10),
				kafkasink.WithReplicationFactor(1))(ctx, t)
		}
	},
}

var pingSource = genericComponent{
	shortLabel: "ps",
	label:      "PingSource",
	kind:       "PingSoure",
	gvr:        pingsource.Gvr(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			pingsource.Install(name, opts...)(ctx, t)
		}
	},
}

var containerSource = genericComponent{
	shortLabel: "cs",
	label:      "ContainerSource",
	kind:       "ContainerSource",
	gvr:        containersource.Gvr(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			containersource.Install(name, opts...)(ctx, t)
		}
	},
}

var apiServerSource = genericComponent{
	shortLabel: "apis",
	label:      "ApiServerSource",
	kind:       "ApiServerSource",
	gvr:        apiserversource.Gvr(),
	install: func(name string, opts ...manifest.CfgFn) feature.StepFn {
		return func(ctx context.Context, t feature.T) {
			//containersource.Install(name, opts...)(ctx, t)
		}
	},
}
