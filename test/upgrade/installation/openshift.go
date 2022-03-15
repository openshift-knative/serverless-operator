package installation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func UpgradeOpenShift(ctx *test.Context) error {
	const clusterVersionName = "version"
	clusterVersion, err := ctx.Clients.ConfigClient.ClusterVersions().Get(context.Background(), clusterVersionName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	var desiredImage string
	if test.Flags.OpenShiftImage != "" {
		desiredImage = test.Flags.OpenShiftImage
		ctx.T.Logf("Desired OpenShift image: %s", desiredImage)
	} else {
		if len(clusterVersion.Status.AvailableUpdates) == 0 {
			return errors.New("no OpenShift upgrades available")
		}
		desiredRelease := clusterVersion.Status.Desired
		// Choose the highest version as the list can be unordered.
		for _, update := range clusterVersion.Status.AvailableUpdates {
			if update.Version > desiredRelease.Version {
				desiredRelease = update
			}
		}
		ctx.T.Logf("Desired OpenShift version: %s", desiredRelease.Version)
		desiredImage = desiredRelease.Image
	}
	clusterVersion.Spec.DesiredUpdate = &configv1.Update{
		Image: desiredImage,
		Force: true,
	}

	if _, err = ctx.Clients.ConfigClient.ClusterVersions().Update(context.Background(),
		clusterVersion, metav1.UpdateOptions{}); err != nil {
		return err
	}

	ctx.T.Logf("Waiting for new cluster version to be ready...")
	clusterVersion, err = WaitForClusterVersionState(ctx, clusterVersionName,
		IsClusterVersionWithImageReady(desiredImage))
	if err != nil {
		return err
	}
	ctx.T.Logf("New cluster version is: %s", clusterVersion.Status.Desired.Version)

	return nil
}

func WaitForClusterVersionState(ctx *test.Context, name string, inState func(s *configv1.ClusterVersion) bool) (*configv1.ClusterVersion, error) {
	var lastState *configv1.ClusterVersion
	var err error
	waitErr := wait.PollImmediate(30*time.Second, 3*time.Hour, func() (bool, error) {
		lastState, err = ctx.Clients.ConfigClient.ClusterVersions().Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			ctx.T.Log("Ignoring error while waiting for ClusterVersion state:", err)
			return false, nil
		}
		return inState(lastState), nil
	})
	if waitErr != nil {
		return lastState, fmt.Errorf("clusterversion %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

type ClusterVersionInStateFunc func(c *configv1.ClusterVersion) bool

func IsClusterVersionWithImageReady(image string) ClusterVersionInStateFunc {
	return func(c *configv1.ClusterVersion) bool {
		for _, h := range c.Status.History {
			if h.Image == image && h.State == configv1.CompletedUpdate {
				return true
			}
		}
		return false
	}
}

func UpgradeEUS(ctx *test.Context) error {
	if updated, err := allMachineConfigPoolsUpdated(ctx); err != nil || !updated {
		return fmt.Errorf("unable to proceed with upgrades: %w", err)
	}

	pauseMachineConfigPool(ctx, true)

	if err := UpgradeOpenShift(ctx); err != nil {
		return fmt.Errorf("failed to upgrade to odd OpenShift release: %w", err)
	}

	if err := UpgradeOpenShift(ctx); err != nil {
		return fmt.Errorf("failed to upgrade to even OpenShift release: %w", err)
	}

	pauseMachineConfigPool(ctx, false)

	if err := wait.PollImmediate(30*time.Second, 3*time.Hour, func() (bool, error) {
		return allMachineConfigPoolsUpdated(ctx)
	}); err != nil {
		return fmt.Errorf("machineconfig pools not updated: %w", err)
	}

	return nil
}

func allMachineConfigPoolsUpdated(ctx *test.Context) (bool, error) {
	poolList, err := ctx.Clients.MachineConfigPool.MachineconfigurationV1().
		MachineConfigPools().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("unable to list machineconfig pools: %w", err)
	}
	for _, mcp := range poolList.Items {
		if !isMachineConfigPoolUpdated(mcp) {
			return false, nil
		}
	}
	return true, nil
}

func isMachineConfigPoolUpdated(mcp machineconfigv1.MachineConfigPool) bool {
	updated := false
	for _, cond := range mcp.Status.Conditions {
		if cond.Type == machineconfigv1.MachineConfigPoolUpdated &&
			cond.Status == v1.ConditionTrue {
			updated = true
		}
	}
	return updated
}

func pauseMachineConfigPool(ctx *test.Context, pause bool) error {
	if _, err := ctx.Clients.Dynamic.
		Resource(machineconfigv1.GroupVersion.WithResource("machineconfigpool")).
		Patch(context.Background(),
			"worker",
			types.MergePatchType,
			[]byte(fmt.Sprintf(`{"spec":{"paused": %t}}`, pause)),
			metav1.PatchOptions{},
		); err != nil {
		return err
	}
	return nil
}
