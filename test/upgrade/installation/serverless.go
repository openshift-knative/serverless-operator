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

	if err := test.ApproveInstallPlan(ctx, installPlan.Name); err != nil {
		return err
	}
	if _, err := test.WaitForClusterServiceVersionState(ctx, csv, test.OperatorsNamespace, test.IsCSVSucceeded); err != nil {
		return err
	}

	servingInStateFunc := v1beta1.IsKnativeServingWithVersionReady(strings.TrimPrefix(test.Flags.ServingVersion, "v"))
	if len(test.Flags.ServingVersion) == 0 {
		servingInStateFunc = v1beta1.IsKnativeServingReady
	}
	knativeServing := test.ServingNamespace
	if _, err := v1beta1.WaitForKnativeServingState(ctx,
		knativeServing,
		knativeServing,
		servingInStateFunc,
	); err != nil {
		return fmt.Errorf("serving upgrade failed: %w", err)
	}

	eventingInStateFunc := v1beta1.IsKnativeEventingWithVersionReady(strings.TrimPrefix(test.Flags.EventingVersion, "v"))
	if len(test.Flags.EventingVersion) == 0 {
		eventingInStateFunc = v1beta1.IsKnativeEventingReady
	}
	knativeEventing := test.EventingNamespace
	if _, err := v1beta1.WaitForKnativeEventingState(ctx,
		knativeEventing,
		knativeEventing,
		eventingInStateFunc,
	); err != nil {
		return fmt.Errorf("eventing upgrade failed: %w", err)
	}

	kafkaInStateFunc := v1alpha1.IsKnativeKafkaWithVersionReady(strings.TrimPrefix(test.Flags.KafkaVersion, "v"))
	if len(test.Flags.KafkaVersion) == 0 {
		kafkaInStateFunc = v1alpha1.IsKnativeKafkaReady
	}
	if _, err := v1alpha1.WaitForKnativeKafkaState(ctx,
		"knative-kafka",
		knativeEventing,
		kafkaInStateFunc,
	); err != nil {
		return fmt.Errorf("knative kafka upgrade failed: %w", err)
	}

	return nil
}

func UpgradeServerless(ctx *test.Context) error {
	return UpgradeServerlessTo(ctx, test.Flags.CSV, test.Flags.CatalogSource, DefaultInstallPlanTimeout)
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

	if err := test.ApproveInstallPlan(ctx, installPlan.Name); err != nil {
		return err
	}

	if _, err := test.WaitForClusterServiceVersionState(ctx, test.Flags.CSVPrevious, test.OperatorsNamespace, test.IsCSVSucceeded); err != nil {
		return err
	}

	knativeServing := test.ServingNamespace
	if _, err := v1beta1.WaitForKnativeServingState(ctx,
		knativeServing,
		knativeServing,
		v1beta1.IsKnativeServingWithVersionReady(strings.TrimPrefix(test.Flags.ServingVersionPrevious, "v")),
	); err != nil {
		return fmt.Errorf("serving downgrade failed: %w", err)
	}

	knativeEventing := test.EventingNamespace
	if _, err := v1beta1.WaitForKnativeEventingState(ctx,
		knativeEventing,
		knativeEventing,
		v1beta1.IsKnativeEventingWithVersionReady(strings.TrimPrefix(test.Flags.EventingVersionPrevious, "v")),
	); err != nil {
		return fmt.Errorf("eventing downgrade failed: %w", err)
	}

	if _, err := v1alpha1.WaitForKnativeKafkaState(ctx,
		"knative-kafka",
		knativeEventing,
		v1alpha1.IsKnativeKafkaWithVersionReady(strings.TrimPrefix(test.Flags.KafkaVersionPrevious, "v")),
	); err != nil {
		return fmt.Errorf("knative kafka downgrade failed: %w", err)
	}

	return nil
}
