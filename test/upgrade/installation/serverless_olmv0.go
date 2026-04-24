package installation

import (
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	"k8s.io/apimachinery/pkg/util/wait"
)

type subscriptionLifecycle struct{}

func (l *subscriptionLifecycle) UpgradeTo(ctx *test.Context, version string, timeout time.Duration) error {
	source := test.Flags.CatalogSource

	if _, err := test.UpdateSubscriptionChannelSource(ctx, test.Flags.Subscription, test.Flags.UpgradeChannel, source); err != nil {
		return err
	}

	installPlan, err := test.WaitForInstallPlan(ctx, test.OperatorsNamespace, version, source, timeout)
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
				test.OperatorsNamespace, version, test.ServerlessOperatorPackage, timeout)
		}
		if err != nil {
			return err
		}
	}

	if err := test.ApproveInstallPlan(ctx, installPlan.Name); err != nil {
		return err
	}
	if _, err := test.WaitForClusterServiceVersionState(ctx, version, test.OperatorsNamespace, test.IsCSVSucceeded); err != nil {
		return err
	}

	return WaitForKnativeComponentsReady(ctx,
		test.Flags.ServingVersion, test.Flags.EventingVersion, test.Flags.KafkaVersion)
}

func (l *subscriptionLifecycle) Upgrade(ctx *test.Context) error {
	return l.UpgradeTo(ctx, test.Flags.CSV, DefaultInstallPlanTimeout)
}

func (l *subscriptionLifecycle) Downgrade(ctx *test.Context) error {
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

	return WaitForKnativeComponentsReady(ctx,
		test.Flags.ServingVersionPrevious, test.Flags.EventingVersionPrevious, test.Flags.KafkaVersionPrevious)
}
