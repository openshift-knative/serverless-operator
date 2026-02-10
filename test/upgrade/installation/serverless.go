package installation

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/v1alpha1"
	"github.com/openshift-knative/serverless-operator/test/v1beta1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	DefaultInstallPlanTimeout = 15 * time.Minute
)

func UpgradeServerlessTo(ctx *test.Context, csv, source string, timeout time.Duration) error {
	ctx.T.Logf("🔄 Starting Serverless upgrade to CSV: %s (source: %s, channel: %s)", csv, source, test.Flags.UpgradeChannel)
	ctx.T.Logf("   Target versions - Serving: %s, Eventing: %s, Kafka: %s",
		test.Flags.ServingVersion, test.Flags.EventingVersion, test.Flags.KafkaVersion)

	if _, err := test.UpdateSubscriptionChannelSource(ctx, test.Flags.Subscription, test.Flags.UpgradeChannel, source); err != nil {
		return err
	}

	installPlan, err := test.WaitForInstallPlan(ctx, test.OperatorsNamespace, csv, source, timeout)
	if err != nil {
		if !wait.Interrupted(err) {
			return err
		}
		if source != test.ServerlessOperatorPackage {
			// InstallPlan not found in the original catalog source, try the one that was just built.
			if _, err := test.UpdateSubscriptionChannelSource(ctx,
				test.Flags.Subscription, test.Flags.UpgradeChannel, test.ServerlessOperatorPackage); err != nil {
				return err
			}
			installPlan, err = test.WaitForInstallPlan(ctx,
				test.OperatorsNamespace, csv, test.ServerlessOperatorPackage, timeout)
		}
		if err != nil {
			return err
		}
	}

	ctx.T.Logf("   Approving InstallPlan: %s", installPlan.Name)
	if err := test.ApproveInstallPlan(ctx, installPlan.Name); err != nil {
		return err
	}
	if _, err := test.WaitForClusterServiceVersionState(ctx, csv, test.OperatorsNamespace, test.IsCSVSucceeded); err != nil {
		return err
	}
	ctx.T.Logf("✅ CSV %s is now in Succeeded state", csv)

	servingVersion := strings.TrimPrefix(test.Flags.ServingVersion, "v")
	servingInStateFunc := v1beta1.IsKnativeServingWithVersionReady(servingVersion)
	if len(test.Flags.ServingVersion) == 0 {
		servingInStateFunc = v1beta1.IsKnativeServingReady
		servingVersion = "<latest>"
	}
	knativeServing := test.ServingNamespace
	ctx.T.Logf("   Waiting for KnativeServing to reach version: %s", servingVersion)
	if _, err := v1beta1.WaitForKnativeServingState(ctx,
		knativeServing,
		knativeServing,
		servingInStateFunc,
	); err != nil {
		return fmt.Errorf("serving upgrade failed: %w", err)
	}
	ctx.T.Logf("✅ KnativeServing is now at version: %s", servingVersion)

	eventingVersion := strings.TrimPrefix(test.Flags.EventingVersion, "v")
	eventingInStateFunc := v1beta1.IsKnativeEventingWithVersionReady(eventingVersion)
	if len(test.Flags.EventingVersion) == 0 {
		eventingInStateFunc = v1beta1.IsKnativeEventingReady
		eventingVersion = "<latest>"
	}
	knativeEventing := test.EventingNamespace
	ctx.T.Logf("   Waiting for KnativeEventing to reach version: %s", eventingVersion)
	if _, err := v1beta1.WaitForKnativeEventingState(ctx,
		knativeEventing,
		knativeEventing,
		eventingInStateFunc,
	); err != nil {
		return fmt.Errorf("eventing upgrade failed: %w", err)
	}
	ctx.T.Logf("✅ KnativeEventing is now at version: %s", eventingVersion)

	kafkaVersion := strings.TrimPrefix(test.Flags.KafkaVersion, "v")
	kafkaInStateFunc := v1alpha1.IsKnativeKafkaWithVersionReady(kafkaVersion)
	if len(test.Flags.KafkaVersion) == 0 {
		kafkaInStateFunc = v1alpha1.IsKnativeKafkaReady
		kafkaVersion = "<latest>"
	}
	ctx.T.Logf("   Waiting for KnativeKafka to reach version: %s", kafkaVersion)
	if _, err := v1alpha1.WaitForKnativeKafkaState(ctx,
		"knative-kafka",
		knativeEventing,
		kafkaInStateFunc,
	); err != nil {
		return fmt.Errorf("knative kafka upgrade failed: %w", err)
	}
	ctx.T.Logf("✅ KnativeKafka is now at version: %s", kafkaVersion)

	ctx.T.Logf("✅ Serverless upgrade completed successfully to CSV: %s", csv)
	return nil
}

func UpgradeServerless(ctx *test.Context) error {
	return UpgradeServerlessTo(ctx, test.Flags.CSV, test.Flags.CatalogSource, DefaultInstallPlanTimeout)
}

func DowngradeServerless(ctx *test.Context) error {
	const subscription = "serverless-operator"

	ctx.T.Logf("🔄 Starting Serverless downgrade to CSV: %s", test.Flags.CSVPrevious)
	ctx.T.Logf("   Target versions - Serving: %s, Eventing: %s, Kafka: %s",
		test.Flags.ServingVersionPrevious, test.Flags.EventingVersionPrevious, test.Flags.KafkaVersionPrevious)

	if err := test.DeleteSubscription(ctx, subscription, test.OperatorsNamespace); err != nil {
		return err
	}

	if err := test.DeleteClusterServiceVersion(ctx, test.Flags.CSV, test.OperatorsNamespace); err != nil {
		return err
	}

	if err := test.WaitForServerlessOperatorsDeleted(ctx); err != nil {
		return err
	}

	// Ensure complete clean up to prevent https://issues.redhat.com/browse/SRVCOM-2203
	if err := test.DeleteNamespace(ctx, test.OperatorsNamespace); err != nil {
		return err
	}

	if _, err := test.CreateNamespace(ctx, test.OperatorsNamespace); err != nil {
		return err
	}

	if _, err := test.CreateOperatorGroup(ctx, "serverless", test.OperatorsNamespace); err != nil {
		return err
	}

	if _, err := test.CreateSubscription(ctx, subscription, test.Flags.CSVPrevious); err != nil {
		return err
	}

	installPlan, err := test.WaitForInstallPlan(ctx, test.OperatorsNamespace, test.Flags.CSVPrevious, test.Flags.CatalogSource, DefaultInstallPlanTimeout)
	if err != nil {
		return err
	}

	ctx.T.Logf("   Approving InstallPlan: %s", installPlan.Name)
	if err := test.ApproveInstallPlan(ctx, installPlan.Name); err != nil {
		return err
	}

	if _, err := test.WaitForClusterServiceVersionState(ctx, test.Flags.CSVPrevious, test.OperatorsNamespace, test.IsCSVSucceeded); err != nil {
		return err
	}
	ctx.T.Logf("✅ CSV %s is now in Succeeded state", test.Flags.CSVPrevious)

	knativeServing := test.ServingNamespace
	servingVersion := strings.TrimPrefix(test.Flags.ServingVersionPrevious, "v")
	servingInStateFunc := v1beta1.IsKnativeServingWithVersionReady(servingVersion)
	ctx.T.Logf("   Waiting for KnativeServing to reach version: %s", servingVersion)
	if _, err := v1beta1.WaitForKnativeServingState(ctx,
		knativeServing,
		knativeServing,
		servingInStateFunc,
	); err != nil {
		return fmt.Errorf("expected ready KnativeServing at version %s: %w", servingVersion, err)
	}
	ctx.T.Logf("✅ KnativeServing is now at version: %s", servingVersion)

	knativeEventing := test.EventingNamespace
	eventingVersion := strings.TrimPrefix(test.Flags.EventingVersionPrevious, "v")
	eventingInStateFunc := v1beta1.IsKnativeEventingWithVersionReady(eventingVersion)
	ctx.T.Logf("   Waiting for KnativeEventing to reach version: %s", eventingVersion)
	if _, err := v1beta1.WaitForKnativeEventingState(ctx,
		knativeEventing,
		knativeEventing,
		eventingInStateFunc,
	); err != nil {
		return fmt.Errorf("expected ready KnativeEventing at version %s: %w", eventingVersion, err)
	}
	ctx.T.Logf("✅ KnativeEventing is now at version: %s", eventingVersion)

	knativeKafkaVersion := strings.TrimPrefix(test.Flags.KafkaVersionPrevious, "v")
	kafkaInStateFunc := v1alpha1.IsKnativeKafkaWithVersionReady(knativeKafkaVersion)
	ctx.T.Logf("   Waiting for KnativeKafka to reach version: %s", knativeKafkaVersion)
	if _, err := v1alpha1.WaitForKnativeKafkaState(ctx,
		"knative-kafka",
		knativeEventing,
		kafkaInStateFunc,
	); err != nil {
		return fmt.Errorf("expected ready KnativeKafka at version %s: %w", knativeKafkaVersion, err)
	}
	ctx.T.Logf("✅ KnativeKafka is now at version: %s", knativeKafkaVersion)

	ctx.T.Logf("✅ Serverless downgrade completed successfully to CSV: %s", test.Flags.CSVPrevious)
	return nil
}
