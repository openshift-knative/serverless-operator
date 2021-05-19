package serving

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestDefaultIngressClass(t *testing.T) {
	cases := []struct {
		name     string
		in       *v1alpha1.KnativeServing
		expected string
	}{{
		name:     "unset",
		in:       &v1alpha1.KnativeServing{},
		expected: kourierIngressClassName,
	}, {
		name: "all disabled",
		in: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				Ingress: &v1alpha1.IngressConfigs{},
			},
		},
		expected: kourierIngressClassName,
	}, {
		name: "istio enabled",
		in: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				Ingress: &v1alpha1.IngressConfigs{
					Istio: v1alpha1.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: istioIngressClassName,
	}, {
		name: "kourier and istio enabled",
		in: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				Ingress: &v1alpha1.IngressConfigs{
					Kourier: v1alpha1.KourierIngressConfiguration{
						Enabled: true,
					},
					Istio: v1alpha1.IstioIngressConfiguration{
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
