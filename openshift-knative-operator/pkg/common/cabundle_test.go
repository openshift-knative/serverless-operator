package common

import (
	"path/filepath"
	"testing"

	"k8s.io/client-go/kubernetes/scheme"

	util "knative.dev/operator/pkg/reconciler/common/testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestApplyCABundlesTransform(t *testing.T) {
	for _, tt := range []struct {
		name     string
		actual   corev1.PodSpec
		expected corev1.PodSpec
	}{{
		name:   "SSL_cert_injectior",
		actual: *podSpecable(t),
		expected: *podSpecable(t,
			withEnvs(
				corev1.EnvVar{
					Name:  "SSL_CERT_DIR",
					Value: "/ocp-serverless-custom-certs:/etc/pki/tls/certs",
				},
			),
			withVolumes(corev1.Volume{
				Name: TrustedCAConfigMapVolume,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: TrustedCAConfigMapName},
						Items: []corev1.KeyToPath{
							{
								Key:  TrustedCAKey,
								Path: TrustedCAKey,
							},
						},
					},
				},
			}),
			withVolumeMounts(
				corev1.VolumeMount{
					Name:      TrustedCAConfigMapVolume,
					MountPath: filepath.Join("/ocp-serverless-custom-certs", TrustedCAKey),
					ReadOnly:  true,
				}),
		),
	}} {
		t.Run(tt.name, func(t *testing.T) {

			caBundleTransform := ApplyCABundlesTransform()

			// test for deployment
			unstructuredDeployment := util.MakeUnstructured(t, util.MakeDeployment(tt.name, tt.actual))
			caBundleTransform(&unstructuredDeployment)
			deployment := &appsv1.Deployment{}
			err := scheme.Scheme.Convert(&unstructuredDeployment, deployment, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, deployment.Spec.Template.Spec, tt.expected)

			// test for statefulset
			unstructuredStatefulset := util.MakeUnstructured(t, util.MakeStatefulSet(tt.name, tt.actual))
			caBundleTransform(&unstructuredStatefulset)
			statefulset := &appsv1.StatefulSet{}
			err = scheme.Scheme.Convert(&unstructuredStatefulset, statefulset, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, statefulset.Spec.Template.Spec, tt.expected)
		})
	}
}

type podSpecableModifier func(*corev1.PodSpec)

func podSpecable(_t *testing.T, modifiers ...podSpecableModifier) *corev1.PodSpec {
	podSpecable := &corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:  "default-container",
			Image: "default-image",
		}},
	}

	for _, modifier := range modifiers {
		modifier(podSpecable)
	}

	return podSpecable
}

func withEnvs(envs ...corev1.EnvVar) func(*corev1.PodSpec) {
	return func(ps *corev1.PodSpec) {
		for i, c := range ps.Containers {
			c.Env = append(c.Env, envs...)
			ps.Containers[i] = c
		}
	}
}

func withVolumes(volumes ...corev1.Volume) func(*corev1.PodSpec) {
	return func(ps *corev1.PodSpec) {
		ps.Volumes = append(ps.Volumes, volumes...)
	}
}

func withVolumeMounts(volumeMounts ...corev1.VolumeMount) func(*corev1.PodSpec) {
	return func(ps *corev1.PodSpec) {
		for i, c := range ps.Containers {
			c.VolumeMounts = append(c.VolumeMounts, volumeMounts...)
			ps.Containers[i] = c
		}
	}
}
