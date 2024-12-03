package serving

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestDefaultToOneAsPodDisruptionBudget(t *testing.T) {
	tests := []struct {
		name     string
		ks       *operatorv1beta1.KnativeServing
		expected []base.PodDisruptionBudgetOverride
	}{
		{
			name: "no overrides",
			ks: &operatorv1beta1.KnativeServing{
				Spec: operatorv1beta1.KnativeServingSpec{},
			},
			expected: []base.PodDisruptionBudgetOverride{
				{
					Name: "activator-pdb",
					PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
						MinAvailable: ptr.To(intstr.IntOrString{Type: intstr.Int, IntVal: 1}),
					},
				},
				{
					Name: "webhook-pdb",
					PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
						MinAvailable: ptr.To(intstr.IntOrString{Type: intstr.Int, IntVal: 1}),
					},
				},
			},
		},
		{
			name: "with existing overrides",
			ks: &operatorv1beta1.KnativeServing{
				Spec: operatorv1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{
						PodDisruptionBudgetOverride: []base.PodDisruptionBudgetOverride{
							{
								Name: "activator-pdb",
								PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
									MinAvailable: ptr.To(intstr.IntOrString{Type: intstr.Int, IntVal: 2}),
								},
							},
						},
					},
				},
			},
			expected: []base.PodDisruptionBudgetOverride{
				{
					Name: "activator-pdb",
					PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
						MinAvailable: ptr.To(intstr.IntOrString{Type: intstr.Int, IntVal: 2}),
					},
				},
				{
					Name: "webhook-pdb",
					PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
						MinAvailable: ptr.To(intstr.IntOrString{Type: intstr.Int, IntVal: 1}),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultToOneAsPodDisruptionBudget(tt.ks)
			if diff := cmp.Diff(tt.expected, tt.ks.Spec.PodDisruptionBudgetOverride); diff != "" {
				t.Errorf("PodDisruptionBudgetOverride mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
