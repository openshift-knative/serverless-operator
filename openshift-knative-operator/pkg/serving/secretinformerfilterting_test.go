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
		name: "default overrides",
		in:   &operatorv1beta1.KnativeServing{},
		expected: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeploymentOverride: []base.WorkloadOverride{
						{
							Name: "net-istio-controller",
							Env: []base.EnvRequirementsOverride{
								{
									Container: "controller",
									EnvVars: []corev1.EnvVar{{
										Name:  EnableSecretInformerFilteringByCertUIDEnv,
										Value: "true",
									}},
								}},
						},
						{
							Name: "net-kourier-controller",
							Env: []base.EnvRequirementsOverride{
								{
									Container: "controller",
									EnvVars: []corev1.EnvVar{{
										Name:  EnableSecretInformerFilteringByCertUIDEnv,
										Value: "true",
									}},
								}},
						},
					},
				}},
		},
		shouldAddLabelToSecret: true,
	}, {
		name: "partial overrides",
		in: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeploymentOverride: []base.WorkloadOverride{
						{
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
						},
					},
				}},
		},
		expected: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeploymentOverride: []base.WorkloadOverride{
						{
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
						},
						{
							Name: "net-istio-controller",
							Env: []base.EnvRequirementsOverride{
								{
									Container: "controller",
									EnvVars: []corev1.EnvVar{{
										Name:  EnableSecretInformerFilteringByCertUIDEnv,
										Value: "true",
									}},
								}},
						},
					},
				}},
		},
		shouldAddLabelToSecret: true,
	}, {
		name: "no overrides",
		in: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeploymentOverride: []base.WorkloadOverride{
						{
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
						},
						{
							Name: "net-istio-controller",
							Env: []base.EnvRequirementsOverride{
								{
									Container: "controller",
									EnvVars: []corev1.EnvVar{{
										Name:  EnableSecretInformerFilteringByCertUIDEnv,
										Value: "false",
									}},
								}},
						},
					},
				}},
		},
		expected: &operatorv1beta1.KnativeServing{
			Spec: operatorv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeploymentOverride: []base.WorkloadOverride{
						{
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
						},
						{
							Name: "net-istio-controller",
							Env: []base.EnvRequirementsOverride{
								{
									Container: "controller",
									EnvVars: []corev1.EnvVar{{
										Name:  EnableSecretInformerFilteringByCertUIDEnv,
										Value: "false",
									}},
								}},
						},
					},
				}},
		},
		shouldAddLabelToSecret: false,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tf := enableSecretInformerFiltering(c.in)
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
