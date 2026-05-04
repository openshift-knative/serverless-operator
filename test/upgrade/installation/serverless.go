package installation

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/v1alpha1"
	"github.com/openshift-knative/serverless-operator/test/v1beta1"
)

const (
	DefaultInstallPlanTimeout = 15 * time.Minute
)

type ServerlessLifecycle interface {
	Upgrade(ctx *test.Context) error
	UpgradeTo(ctx *test.Context, version string, timeout time.Duration) error
	Downgrade(ctx *test.Context) error
}

func NewServerlessLifecycle(olmVersion string) ServerlessLifecycle {
	if olmVersion == "v1" {
		return &clusterExtensionLifecycle{}
	}
	return &subscriptionLifecycle{}
}

func WaitForKnativeComponentsReady(ctx *test.Context, servingVersion, eventingVersion, kafkaVersion string) error {
	servingInStateFunc := v1beta1.IsKnativeServingWithVersionReady(strings.TrimPrefix(servingVersion, "v"))
	if len(servingVersion) == 0 {
		servingInStateFunc = v1beta1.IsKnativeServingReady
	}
	if _, err := v1beta1.WaitForKnativeServingState(ctx,
		test.ServingNamespace,
		test.ServingNamespace,
		servingInStateFunc,
	); err != nil {
		return fmt.Errorf("expected ready KnativeServing at version %s: %w", servingVersion, err)
	}

	eventingInStateFunc := v1beta1.IsKnativeEventingWithVersionReady(strings.TrimPrefix(eventingVersion, "v"))
	if len(eventingVersion) == 0 {
		eventingInStateFunc = v1beta1.IsKnativeEventingReady
	}
	if _, err := v1beta1.WaitForKnativeEventingState(ctx,
		test.EventingNamespace,
		test.EventingNamespace,
		eventingInStateFunc,
	); err != nil {
		return fmt.Errorf("expected ready KnativeEventing at version %s: %w", eventingVersion, err)
	}

	kafkaInStateFunc := v1alpha1.IsKnativeKafkaWithVersionReady(strings.TrimPrefix(kafkaVersion, "v"))
	if len(kafkaVersion) == 0 {
		kafkaInStateFunc = v1alpha1.IsKnativeKafkaReady
	}
	if _, err := v1alpha1.WaitForKnativeKafkaState(ctx,
		"knative-kafka",
		test.EventingNamespace,
		kafkaInStateFunc,
	); err != nil {
		return fmt.Errorf("expected ready KnativeKafka at version %s: %w", kafkaVersion, err)
	}

	return nil
}
