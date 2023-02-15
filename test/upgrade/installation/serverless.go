package installation

import (
	"context"
	"fmt"
	"strings"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/v1alpha1"
	"github.com/openshift-knative/serverless-operator/test/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func UpgradeServerless(ctx *test.Context) error {
	sources := strings.Split(strings.Trim(test.Flags.CatalogSource, ","), ",")
	csvs := strings.Split(strings.Trim(test.Flags.CSV, ","), ",")
	// For each CSV there must be a corresponding catalog source.
	if len(sources) != len(csvs) {
		return fmt.Errorf("the number of operator sources and CSVs for upgrades must match")
	}
	for i, csv := range csvs {
		source := sources[i]
		if _, err := test.UpdateSubscriptionChannelSource(ctx, test.Flags.Subscription, test.Flags.UpgradeChannel, source); err != nil {
			return err
		}

		installPlan, err := test.WaitForInstallPlan(ctx, test.OperatorsNamespace, csv, source)
		if err != nil {
			return err
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
		knativeServing := "knative-serving"
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
		knativeEventing := "knative-eventing"
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

	// Ensure complete clean up to prevent https://issues.redhat.com/browse/SRVCOM-2203
	if err := test.DeleteNamespace(ctx, test.OperatorsNamespace); err != nil {
		return err
	}

	// If we are on OCP 4.8 we need to apply the workaround in https://access.redhat.com/solutions/6992396.
	// Currently, we only test in 4.8 (1.21) and 4.11+ (1.24+). Latest versions (eg. 4.11+) have a fix for this so no need to patch the crds,
	// but we do it anyway for supported versions up to 4.10.
	if err := common.CheckMinimumKubeVersion(ctx.Clients.Kube.Discovery(), "1.23.0"); err != nil {
		for _, name := range crds {
			if err := setWebookStrategyToNone(ctx, name); err != nil {
				return err
			}
		}
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
	if _, err := v1beta1.WaitForKnativeServingState(ctx,
		knativeServing,
		knativeServing,
		v1beta1.IsKnativeServingWithVersionReady(strings.TrimPrefix(test.Flags.ServingVersionPrevious, "v")),
	); err != nil {
		return fmt.Errorf("serving downgrade failed: %w", err)
	}

	knativeEventing := "knative-eventing"
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

func setWebookStrategyToNone(ctx *test.Context, name string) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		crd, err := ctx.Clients.APIExtensionClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.ConversionStrategyType("None")}
		_, err = ctx.Clients.APIExtensionClient.ApiextensionsV1().CustomResourceDefinitions().Update(context.Background(), crd, metav1.UpdateOptions{})
		return err
	})
}
