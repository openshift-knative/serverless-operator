package installation

import (
	"context"
	"fmt"
	"strings"

	"github.com/openshift-knative/serverless-operator/test"
	v1b1test "github.com/openshift-knative/serverless-operator/test/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	crds := []string{"knativeservings.operator.knative.dev", "knativeeventings.operator.knative.dev"}

	if err := test.DeleteSubscription(ctx, subscription, test.OperatorsNamespace); err != nil {
		return err
	}

	if err := test.DeleteClusterServiceVersion(ctx, test.Flags.CSV, test.OperatorsNamespace); err != nil {
		return err
	}

	if err := test.WaitForServerlessOperatorsDeleted(ctx); err != nil {
		return err
	}

	for _, name := range crds {
		if err := moveCRDsToAlpha(ctx, name); err != nil {
			return err
		}
	}

	// Do an empty patch to trigger etcd storage cleanup
	if _, err := ctx.Clients.OperatorAlpha.KnativeServings("knative-serving").Patch(context.Background(), "knative-serving", types.MergePatchType, []byte("{}"), metav1.PatchOptions{}, ""); err != nil {
		return err
	}
	if _, err := ctx.Clients.OperatorAlpha.KnativeEventings("knative-eventing").Patch(context.Background(), "knative-eventing", types.MergePatchType, []byte("{}"), metav1.PatchOptions{}, ""); err != nil {
		return err
	}

	// Remove stored version from status subresource in CRDs
	for _, name := range crds {
		if err := setStorageToAlpha(ctx, name); err != nil {
			return err
		}
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
	if _, err := v1b1test.WaitForKnativeServingStateAlpha(ctx,
		knativeServing,
		knativeServing,
		v1b1test.IsKnativeServingWithVersionReadyAlpha(strings.TrimPrefix(test.Flags.ServingVersionPrevious, "v")),
	); err != nil {
		return fmt.Errorf("serving downgrade failed: %w", err)
	}

	knativeEventing := "knative-eventing"
	if _, err := v1b1test.WaitForKnativeEventingStateAlpha(ctx,
		knativeEventing,
		knativeEventing,
		v1b1test.IsKnativeEventingWithVersionReadyAlpha(strings.TrimPrefix(test.Flags.EventingVersionPrevious, "v")),
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

func moveCRDsToAlpha(ctx *test.Context, name string) error {
	crd, err := ctx.Clients.APIExtensionClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for i, v := range crd.Spec.Versions {
		if v.Name == "v1beta1" {
			crd.Spec.Versions[i].Served = false
			crd.Spec.Versions[i].Storage = false
		}

		if v.Name == "v1alpha1" {
			crd.Spec.Versions[i].Served = true
			crd.Spec.Versions[i].Storage = true
		}
	}
	crd.Spec.Conversion = &apiextension.CustomResourceConversion{Strategy: apiextension.ConversionStrategyType("None")}
	_, err = ctx.Clients.APIExtensionClient.ApiextensionsV1().CustomResourceDefinitions().Update(context.Background(), crd, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func setStorageToAlpha(ctx *test.Context, name string) error {
	crd, err := ctx.Clients.APIExtensionClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	oldStoredVersions := crd.Status.StoredVersions
	newStoredVersions := make([]string, 0, len(oldStoredVersions))
	for _, stored := range oldStoredVersions {
		if stored != "v1beta1" {
			newStoredVersions = append(newStoredVersions, stored)
		}
	}
	crd.Status.StoredVersions = newStoredVersions
	_, err = ctx.Clients.APIExtensionClient.ApiextensionsV1().CustomResourceDefinitions().UpdateStatus(context.Background(), crd, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
