package serving

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestSecretInformerFilteringOverride(t *testing.T) {
	cases := []struct {
		name                                string
		in                                  *operatorv1beta1.KnativeServing
		expected                            *operatorv1beta1.KnativeServing
		shouldEnableSecretInformerFiltering bool
	}{{
		name: "by default no overrides, enabled secret informer filtering, with kourier enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Kourier: base.KourierIngressConfiguration{
				Enabled: true,
			}}
		}),
		expected: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Kourier: base.KourierIngressConfiguration{
				Enabled: true,
			}}
		}),
		shouldEnableSecretInformerFiltering: true,
	}, {
		name: "by default no overrides, enabled secret informer filtering, with istio enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
		}),
		expected: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
		}),
		shouldEnableSecretInformerFiltering: true,
	}, {
		name: "disabled secret informer filtering with kourier enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Kourier: base.KourierIngressConfiguration{
				Enabled: true,
			}}
			ks.Spec.DeploymentOverride = append(ks.Spec.DeploymentOverride, base.WorkloadOverride{
				Name: "net-kourier-controller",
				Env: []base.EnvRequirementsOverride{
					{
						Container: "controller",
						EnvVars: []corev1.EnvVar{{
							Name:  EnableSecretInformerFilteringByCertUIDEnv,
							Value: "false",
						}, {
							Name:  "foo",
							Value: "foo",
						}},
					}},
			})
		}),
		expected: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Kourier: base.KourierIngressConfiguration{
				Enabled: true,
			}}
			ks.Spec.DeploymentOverride = append(ks.Spec.DeploymentOverride, base.WorkloadOverride{
				Name: "net-kourier-controller",
				Env: []base.EnvRequirementsOverride{
					{
						Container: "controller",
						EnvVars: []corev1.EnvVar{{
							Name:  EnableSecretInformerFilteringByCertUIDEnv,
							Value: "false",
						}, {
							Name:  "foo",
							Value: "foo",
						}},
					}},
			})
		}),
		shouldEnableSecretInformerFiltering: false,
	}, {
		name: "disabled secret informer filtering with istio enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Spec.DeploymentOverride = append(ks.Spec.DeploymentOverride, base.WorkloadOverride{
				Name: "net-istio-controller",
				Env: []base.EnvRequirementsOverride{
					{
						Container: "controller",
						EnvVars: []corev1.EnvVar{{
							Name:  EnableSecretInformerFilteringByCertUIDEnv,
							Value: "false",
						}, {
							Name:  "foo",
							Value: "foo",
						}},
					}},
			})
		}),
		expected: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Spec.DeploymentOverride = append(ks.Spec.DeploymentOverride, base.WorkloadOverride{
				Name: "net-istio-controller",
				Env: []base.EnvRequirementsOverride{
					{
						Container: "controller",
						EnvVars: []corev1.EnvVar{{
							Name:  EnableSecretInformerFilteringByCertUIDEnv,
							Value: "false",
						}, {
							Name:  "foo",
							Value: "foo",
						}},
					}},
			})
		}),
		shouldEnableSecretInformerFiltering: false,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tfs := enableSecretInformerFilteringTransformers(c.in)
			if c.shouldEnableSecretInformerFiltering {
				if len(tfs) != 2 {
					t.Errorf("There should be two transformers, but got %d", len(tfs))
				}
				if tfs[1] == nil {
					t.Errorf("Secret informer filtering transformer should not be nil")
				}
			} else if !c.shouldEnableSecretInformerFiltering && tfs != nil {
				t.Errorf("Secret informer filtering transformers should be nil")
			}
			if !cmp.Equal(c.in, c.expected) {
				t.Errorf("Resource was not as expected:\n%s", cmp.Diff(c.in, c.expected))
			}
		})
	}
}
