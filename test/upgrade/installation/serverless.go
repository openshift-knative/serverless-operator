package installation

import (
	"context"
	"fmt"
	"strings"

	kafkav1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/test"
	v1a1test "github.com/openshift-knative/serverless-operator/test/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		return fmt.Errorf("serving upgrade failed: %w", err)
	}

	knativeEventing := "knative-eventing"
	if _, err := v1a1test.WaitForKnativeEventingState(ctx,
		knativeEventing,
		knativeEventing,
		v1a1test.IsKnativeEventingWithVersionReady(strings.TrimPrefix(test.Flags.EventingVersion, "v")),
	); err != nil {
		return fmt.Errorf("eventing upgrade failed: %w", err)
	}

	if _, err := v1a1test.WaitForKnativeKafkaState(ctx,
		"knative-kafka",
		knativeEventing,
		v1a1test.IsKnativeKafkaWithVersionReady(strings.TrimPrefix(test.Flags.KafkaVersion, "v")),
	); err != nil {
		return fmt.Errorf("knative kafka upgrade failed: %w", err)
	}

	return nil
}

func EnableKafkaBroker(ctx *test.Context) error {
	if _, err := ctx.Clients.Dynamic.
		Resource(kafkav1alpha1.SchemeGroupVersion.WithResource("knativekafkas")).
		Namespace("knative-eventing").
		Patch(context.Background(),
			"knative-kafka",
			types.MergePatchType,
			[]byte(`{"spec":{"broker":{"enabled": true}}}`),
			metav1.PatchOptions{},
		); err != nil {
		return err
	}
	return nil
}
