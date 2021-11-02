package installation

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

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

func WaitForPodsWithImage(ctx *test.Context, namespace string, podSelector, containerName, expectedImage string) error {
	if waitErr := wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
		podList, err := ctx.Clients.Kube.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: podSelector})
		if err != nil {
			return false, err
		}
		for _, pod := range podList.Items {
			for _, c := range pod.Spec.Containers {
				if c.Name == containerName && c.Image != expectedImage {
					return false, nil
				}
			}
		}
		return true, nil
	}); waitErr != nil {
		return fmt.Errorf("containers %s in pods with label selector %s do not have the expected image: %w", containerName, podSelector, waitErr)
	}
	return nil
}
