package serving

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestDefaultIngressClass(t *testing.T) {
	cases := []struct {
		name     string
		in       *operatorv1beta1.KnativeServing
		expected string
	}{{
		name:     "unset",
		in:       &operatorv1beta1.KnativeServing{},
		expected: kourierIngressClassName,
	}, {
		name: "all disabled",
		in: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				Ingress: &operatorv1beta1.IngressConfigs{},
			},
		},
		expected: kourierIngressClassName,
	}, {
		name: "istio enabled",
		in: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				Ingress: &operatorv1beta1.IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: istioIngressClassName,
	}, {
		name: "kourier and istio enabled",
		in: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				Ingress: &operatorv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
					Istio: base.IstioIngressConfiguration{
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
