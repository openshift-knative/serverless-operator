package brokerconfig

import (
	"context"
	"embed"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/inmemorychannel"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/eventing-kafka-broker/test/rekt/resources/kafkachannel"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/manifest"
)

//go:embed *.yaml
var yaml embed.FS

const defaultKafkaBrokerBootstrap = "my-cluster-kafka-bootstrap.kafka:9092"
const defaultKafkaPartitions = 10
const defaultKafkaReplicationFactor = 3

func GVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
}

// Install will create a Broker ConfigMap, augmented with the config fn options.
func Install(name string, opts ...manifest.CfgFn) feature.StepFn {
	cfg := map[string]interface{}{
		"name": name,
	}
	for _, fn := range opts {
		fn(cfg)
	}
	return func(ctx context.Context, t feature.T) {
		if _, err := manifest.InstallYamlFS(ctx, yaml, cfg); err != nil {
			t.Fatal(err)
		}
	}
}

func WithKafkaChannelMTBroker(kafkaChannelOpts ...manifest.CfgFn) manifest.CfgFn {
	return func(cfg map[string]interface{}) {
		cfg["kafkaChannel"] = map[string]interface{}{
			"version":           kafkachannel.GVR().Version,
			"numPartitions":     defaultKafkaPartitions,
			"replicationFactor": defaultKafkaReplicationFactor,
		}

		for _, fn := range kafkaChannelOpts {
			fn(cfg["kafkaChannel"].(map[string]interface{}))
		}
	}
}

func WithInMemoryChannelMTBroker() manifest.CfgFn {
	return func(cfg map[string]interface{}) {
		cfg["inMemoryChannel"] = map[string]interface{}{
			"version": inmemorychannel.GVR().Version,
		}
	}
}

func WithKafkaBroker(kafkaOpts ...manifest.CfgFn) manifest.CfgFn {
	return func(cfg map[string]interface{}) {
		cfg["kafkaBroker"] = map[string]interface{}{
			"bootstrap":         defaultKafkaBrokerBootstrap,
			"replicationFactor": defaultKafkaReplicationFactor,
			"partitions":        defaultKafkaPartitions,
		}

		for _, fn := range kafkaOpts {
			fn(cfg["kafkaBroker"].(map[string]interface{}))
		}
	}
}
