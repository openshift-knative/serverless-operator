package test

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func WaitForInstallPlan(ctx *Context, namespace string, csvName, olmSource string) (*v1alpha1.InstallPlan, error) {
	var plan *v1alpha1.InstallPlan
	if waitErr := wait.PollImmediate(Interval, 15*time.Minute, func() (bool, error) {
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

func installsCSVFromSource(installPlan v1alpha1.InstallPlan, csvName, olmSource string) bool {
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
