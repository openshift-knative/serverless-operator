package serving

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestSecretInformerFitleringOverride(t *testing.T) {
	cases := []struct {
		name                   string
		in                     *operatorv1beta1.KnativeServing
		expected               *operatorv1beta1.KnativeServing
		shouldAddLabelToSecret bool
	}{{
		name: "by default no overrides, disabled secret filtering, with kourier enabled",
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
		shouldAddLabelToSecret: false,
	}, {
		name: "by default no overrides, disabled secret filtering, with istio enabled",
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
		shouldAddLabelToSecret: false,
	}, {
		name: "by default no overrides, deprecated annotation set to true, enabled secret filtering, istio enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "true"}
		}),
		expected: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "true"}
		}),
		shouldAddLabelToSecret: true,
	}, {
		name: "by default no overrides, istio deprecated annotation set to true but has no effect, disabled secret filtering, kourier enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Kourier: base.KourierIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "true"}
		}),
		expected: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Kourier: base.KourierIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "true"}
		}),
		shouldAddLabelToSecret: false,
	}, {
		name: "by default no overrides, istio deprecated annotation set to false, disabled secret filtering, istio enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "false"}
		}),
		expected: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "false"}
		}),
		shouldAddLabelToSecret: false,
	}, {
		name: "by default no overrides, istio deprecated annotation value with bad string, disabled secret filtering, istio enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "wrong"}
		}),
		expected: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "wrong"}
		}),
		shouldAddLabelToSecret: false,
	}, {
		name: "disabled secret filtering with kourier enabled",
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
		shouldAddLabelToSecret: false,
	}, {
		name: "disabled secret filtering with istio enabled",
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
		shouldAddLabelToSecret: false,
	}, {
		name: "deprecated annotation is set to true but env var takes precedence, disabled secret filtering, istio enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "true"}
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
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "true"}
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
		shouldAddLabelToSecret: false,
	}, {
		name: "deprecated annotation is set to false but env var takes precedence, enabled secret filtering, istio enabled",
		in: ks(func(ks *operatorv1beta1.KnativeServing) {
			ks.Spec.Ingress = &operatorv1beta1.IngressConfigs{Istio: base.IstioIngressConfiguration{
				Enabled: true,
			}}
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "false"}
			ks.Spec.DeploymentOverride = append(ks.Spec.DeploymentOverride, base.WorkloadOverride{
				Name: "net-istio-controller",
				Env: []base.EnvRequirementsOverride{
					{
						Container: "controller",
						EnvVars: []corev1.EnvVar{{
							Name:  EnableSecretInformerFilteringByCertUIDEnv,
							Value: "true",
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
			ks.Annotations = map[string]string{secretInformerFilteringAnnotation: "false"}
			ks.Spec.DeploymentOverride = append(ks.Spec.DeploymentOverride, base.WorkloadOverride{
				Name: "net-istio-controller",
				Env: []base.EnvRequirementsOverride{
					{
						Container: "controller",
						EnvVars: []corev1.EnvVar{{
							Name:  EnableSecretInformerFilteringByCertUIDEnv,
							Value: "true",
						}, {
							Name:  "foo",
							Value: "foo",
						}},
					}},
			})
		}),
		shouldAddLabelToSecret: false,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tf := enableSecretInformerFilteringTransformers(c.in)
			if c.shouldAddLabelToSecret {
				if tf == nil {
					t.Errorf("Secret transformer should not be nil")
				}
			}
			if !cmp.Equal(c.in, c.expected) {
				t.Errorf("Resource was not as expected:\n%s", cmp.Diff(c.in, c.expected))
			}
		})
	}
}
