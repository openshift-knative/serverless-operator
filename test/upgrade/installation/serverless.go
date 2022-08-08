package installation

import (
	"fmt"
	"strings"

	"github.com/openshift-knative/serverless-operator/test"
	v1b1test "github.com/openshift-knative/serverless-operator/test/v1beta1"
)

func UpgradeServerless(ctx *test.Context) error {
	if _, err := test.UpdateSubscriptionChannelSource(ctx, test.Flags.Subscription, test.Flags.UpgradeChannel, test.Flags.CatalogSource); err != nil {
		return err
	}

	installPlan, err := test.WaitForInstallPlan(ctx, test.OperatorsNamespace, test.Flags.CSV, test.Flags.CatalogSource)
	if err != nil {
		return err
	}

	if err := test.ApproveInstallPlan(ctx, installPlan.Name); err != nil {
		return err
	}
	if _, err := test.WaitForClusterServiceVersionState(ctx, test.Flags.CSV, test.OperatorsNamespace, test.IsCSVSucceeded); err != nil {
		return err
	}

	knativeServing := "knative-serving"
	if _, err := v1b1test.WaitForKnativeServingState(ctx,
		knativeServing,
		knativeServing,
		v1b1test.IsKnativeServingWithVersionReady(strings.TrimPrefix(test.Flags.ServingVersion, "v")),
	); err != nil {
		return fmt.Errorf("serving upgrade failed: %w", err)
	}

	knativeEventing := "knative-eventing"
	if _, err := v1b1test.WaitForKnativeEventingState(ctx,
		knativeEventing,
		knativeEventing,
		v1b1test.IsKnativeEventingWithVersionReady(strings.TrimPrefix(test.Flags.EventingVersion, "v")),
	); err != nil {
		return fmt.Errorf("eventing upgrade failed: %w", err)
	}

	if _, err := v1b1test.WaitForKnativeKafkaState(ctx,
		"knative-kafka",
		knativeEventing,
		v1b1test.IsKnativeKafkaWithVersionReady(strings.TrimPrefix(test.Flags.KafkaVersion, "v")),
	); err != nil {
		return fmt.Errorf("knative kafka upgrade failed: %w", err)
	}

	return nil
}

func DowngradeServerless(ctx *test.Context) error {
	const subscription = "serverless-operator"

	if err := test.DeleteSubscription(ctx, subscription, test.OperatorsNamespace); err != nil {
		return err
	}

	if err := test.DeleteClusterServiceVersion(ctx, test.Flags.CSV, test.OperatorsNamespace); err != nil {
		return err
	}

	if err := test.WaitForServerlessOperatorsDeleted(ctx); err != nil {
		return err
	}

	if _, err := test.CreateSubscription(ctx, subscription, test.Flags.CSVPrevious); err != nil {
		return err
	}

	installPlan, err := test.WaitForInstallPlan(ctx, test.OperatorsNamespace, test.Flags.CSVPrevious, test.Flags.CatalogSource)
	if err != nil {
		return err
	}

	if err := test.ApproveInstallPlan(ctx, installPlan.Name); err != nil {
		return err
	}

	if _, err := test.WaitForClusterServiceVersionState(ctx, test.Flags.CSVPrevious, test.OperatorsNamespace, test.IsCSVSucceeded); err != nil {
		return err
	}

	knativeServing := "knative-serving"
	if _, err := v1b1test.WaitForKnativeServingState(ctx,
		knativeServing,
		knativeServing,
		v1b1test.IsKnativeServingWithVersionReady(strings.TrimPrefix(test.Flags.ServingVersionPrevious, "v")),
	); err != nil {
		return fmt.Errorf("serving downgrade failed: %w", err)
	}

	knativeEventing := "knative-eventing"
	if _, err := v1b1test.WaitForKnativeEventingState(ctx,
		knativeEventing,
		knativeEventing,
		v1b1test.IsKnativeEventingWithVersionReady(strings.TrimPrefix(test.Flags.EventingVersionPrevious, "v")),
	); err != nil {
		return fmt.Errorf("eventing downgrade failed: %w", err)
	}

	if _, err := v1b1test.WaitForKnativeKafkaState(ctx,
		"knative-kafka",
		knativeEventing,
		v1b1test.IsKnativeKafkaWithVersionReady(strings.TrimPrefix(test.Flags.KafkaVersionPrevious, "v")),
	); err != nil {
		return fmt.Errorf("knative kafka downgrade failed: %w", err)
	}

	return nil
}
