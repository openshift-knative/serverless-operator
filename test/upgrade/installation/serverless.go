package installation

import (
	"strings"

	"github.com/openshift-knative/serverless-operator/test"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
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
	if _, err := v1a1test.WaitForKnativeServingState(ctx,
		knativeServing,
		knativeServing,
		v1a1test.IsKnativeServingWithVersionReady(strings.TrimPrefix(test.Flags.ServingVersion, "v")),
	); err != nil {
		ctx.T.Error("Serving upgrade failed:", err)
	}

	knativeEventing := "knative-eventing"
	if _, err := v1a1test.WaitForKnativeEventingState(ctx,
		knativeEventing,
		knativeEventing,
		v1a1test.IsKnativeEventingWithVersionReady(strings.TrimPrefix(test.Flags.EventingVersion, "v")),
	); err != nil {
		ctx.T.Error("Eventing upgrade failed:", err)
	}

	return nil
}
