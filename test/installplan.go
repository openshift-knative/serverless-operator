package test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/wait"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func WaitForInstallPlan(ctx *Context, namespace string, csvName, olmSource string, timeout time.Duration) (*operatorsv1alpha1.InstallPlan, error) {
	var plan *operatorsv1alpha1.InstallPlan
	if waitErr := wait.PollImmediate(Interval, timeout, func() (bool, error) {
		installPlans, err := ctx.Clients.OLM.OperatorsV1alpha1().InstallPlans(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, installPlan := range installPlans.Items {
			if installsCSVFromSource(installPlan, csvName, olmSource) {
				plan = &installPlan
				return true, nil
			}
		}
		return false, nil
	}); waitErr != nil {
		return plan, fmt.Errorf("installplan for CSV %s and OLM source %s not found: %w", csvName, olmSource, waitErr)
	}
	return plan, nil
}

func installsCSVFromSource(installPlan operatorsv1alpha1.InstallPlan, csvName, olmSource string) bool {
	if installPlan.Status.BundleLookups == nil ||
		len(installPlan.Status.BundleLookups) == 0 ||
		installPlan.Status.BundleLookups[0].CatalogSourceRef == nil ||
		installPlan.Status.BundleLookups[0].CatalogSourceRef.Name != olmSource {
		return false
	}
	for _, name := range installPlan.Spec.ClusterServiceVersionNames {
		if name == csvName {
			return true
		}
	}
	return false
}

func ApproveInstallPlan(ctx *Context, name string) error {
	patch := []byte(`{"spec":{"approved":true}}`)
	_, err := ctx.Clients.OLM.OperatorsV1alpha1().InstallPlans(OperatorsNamespace).
		Patch(context.Background(), name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

func DeleteInstallPlan(ctx *Context, namespace string, csvName, olmSource string) error {
	installPlan, err := WaitForInstallPlan(ctx, namespace, csvName, olmSource, time.Millisecond)
	// Ignore the error when the InstallPlan is already removed.
	if err != nil && !errors.Is(err, wait.ErrWaitTimeout) {
		return err
	}
	err = ctx.Clients.OLM.OperatorsV1alpha1().InstallPlans(namespace).Delete(context.Background(), installPlan.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}
