package serving

import (
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

var servingPDBNames = []string{"activator-pdb", "webhook-pdb"}

// Upstream has a PodDisruptionBudgetOverride with minAvailable: 80% which does not work with
// HighAvailability of two Pods. We need to override this to minAvailable: 1 if the user did not specify
// another value.
func defaultToOneAsPodDisruptionBudget(ks *operatorv1beta1.KnativeServing) {
	overrides := ks.GetSpec().GetPodDisruptionBudgetOverride()

	if overrides == nil {
		overrides = []base.PodDisruptionBudgetOverride{}
	}

	for _, pdbName := range servingPDBNames {
		if !hasOverride(overrides, pdbName) {
			ks.Spec.PodDisruptionBudgetOverride = append(ks.Spec.PodDisruptionBudgetOverride, base.PodDisruptionBudgetOverride{
				Name: pdbName,
				PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
					MinAvailable: ptr.To(intstr.IntOrString{Type: intstr.Int, IntVal: 1}),
				},
			})
		}
	}
}

func hasOverride(overrides []base.PodDisruptionBudgetOverride, pdbName string) bool {
	for _, override := range overrides {
		if override.Name == pdbName {
			return true
		}
	}
	return false
}
