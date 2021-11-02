package serving

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestDefaultIngressClass(t *testing.T) {
	cases := []struct {
		name     string
		in       *operatorv1alpha1.KnativeServing
		expected string
	}{{
		name:     "unset",
		in:       &operatorv1alpha1.KnativeServing{},
		expected: kourierIngressClassName,
	}, {
		name: "all disabled",
		in: &operatorv1alpha1.KnativeServing{
			Spec: operatorv1alpha1.KnativeServingSpec{
				Ingress: &operatorv1alpha1.IngressConfigs{},
			},
		},
		expected: kourierIngressClassName,
	}, {
		name: "istio enabled",
		in: &operatorv1alpha1.KnativeServing{
			Spec: operatorv1alpha1.KnativeServingSpec{
				Ingress: &operatorv1alpha1.IngressConfigs{
					Istio: operatorv1alpha1.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: istioIngressClassName,
	}, {
		name: "kourier and istio enabled",
		in: &operatorv1alpha1.KnativeServing{
			Spec: operatorv1alpha1.KnativeServingSpec{
				Ingress: &operatorv1alpha1.IngressConfigs{
					Kourier: operatorv1alpha1.KourierIngressConfiguration{
						Enabled: true,
					},
					Istio: operatorv1alpha1.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: kourierIngressClassName,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := defaultIngressClass(c.in)
			if !cmp.Equal(got, c.expected) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", got, c.expected, cmp.Diff(got, c.expected))
			}
		})
	}
}
