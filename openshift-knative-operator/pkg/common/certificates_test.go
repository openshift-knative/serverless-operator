package common

import (
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestAddCABundleConfigMapsToVolumes(t *testing.T) {
	type testStructure struct {
		name     string
		input    []corev1.Volume
		expected []corev1.Volume
	}

	tests := []testStructure{
		{
			name:  "Vanilla test without any input volumes",
			input: nil,
			expected: []corev1.Volume{
				{
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
				},
			},
		},
		{
			name: "Check if volumes are appended",
			input: []corev1.Volume{
				{
					Name: "bleh",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "bleh"},
							Items: []corev1.KeyToPath{
								{
									Key:  "bleh",
									Path: "bleh",
								},
							},
						},
					},
				},
			},
			expected: []corev1.Volume{
				{
					Name: "bleh",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "bleh"},
							Items: []corev1.KeyToPath{
								{
									Key:  "bleh",
									Path: "bleh",
								},
							},
						},
					},
				},
				{
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
				},
			},
		},
		{
			name: "Check if duplicate volumes are removed",
			input: []corev1.Volume{
				{
					Name: TrustedCAConfigMapVolume,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "bleh"},
							Items: []corev1.KeyToPath{
								{
									Key:  "bleh",
									Path: "bleh",
								},
							},
						},
					},
				},
			},
			expected: []corev1.Volume{
				{
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
				},
			},
		},
	}

	for _, test := range tests {
		t.Logf("Running test: %v", test.name)
		actualOutput := AddCABundleConfigMapsToVolumes(test.input)
		util.AssertDeepEqual(t, actualOutput, test.expected)
	}
}

func TestAddCABundlesToContainerVolumes(t *testing.T) {
	type testStructure struct {
		name     string
		input    *corev1.Container
		expected *corev1.Container
	}

	defaultSSLCertDir := "/ocp-serverless-custom-certs:/etc/pki/tls/certs"

	tests := []testStructure{
		{
			name:  "Check baseline functionality - default SSL_CERT_DIR value, default volume mounts",
			input: &corev1.Container{},
			expected: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  "SSL_CERT_DIR",
						Value: defaultSSLCertDir,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      TrustedCAConfigMapVolume,
						MountPath: filepath.Join("/ocp-serverless-custom-certs", TrustedCAKey),
						SubPath:   TrustedCAKey,
						ReadOnly:  true,
					},
				},
			},
		},
		{
			name: "Check if duplicates are removed",
			input: &corev1.Container{
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      TrustedCAConfigMapVolume,
						MountPath: "bleh",
						SubPath:   "bleh",
						ReadOnly:  false,
					},
				},
			},
			expected: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  "SSL_CERT_DIR",
						Value: defaultSSLCertDir,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      TrustedCAConfigMapVolume,
						MountPath: filepath.Join("/ocp-serverless-custom-certs", TrustedCAKey),
						SubPath:   TrustedCAKey,
						ReadOnly:  true,
					},
				},
			},
		},
		{
			name: "Check if volume mounts are appended",
			input: &corev1.Container{
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "bleh",
						MountPath: "bleh",
						SubPath:   "bleh",
						ReadOnly:  false,
					},
				},
			},
			expected: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  "SSL_CERT_DIR",
						Value: defaultSSLCertDir,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "bleh",
						MountPath: "bleh",
						SubPath:   "bleh",
						ReadOnly:  false,
					},
					{
						Name:      TrustedCAConfigMapVolume,
						MountPath: filepath.Join("/ocp-serverless-custom-certs", TrustedCAKey),
						SubPath:   TrustedCAKey,
						ReadOnly:  true,
					},
				},
			},
		},
		{
			name: "Check if already existing SSL_CERT_DIR is preserved",
			input: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  "SSL_CERT_DIR",
						Value: "/existing/ssl/cert/dir",
					},
				},
			},
			expected: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  "SSL_CERT_DIR",
						Value: "/existing/ssl/cert/dir",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      TrustedCAConfigMapVolume,
						MountPath: filepath.Join("/existing/ssl/cert/dir", TrustedCAKey),
						SubPath:   TrustedCAKey,
						ReadOnly:  true,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Logf("Running test: %v", test.name)
		AddCABundlesToContainerVolumes(test.input)
		util.AssertDeepEqual(t, test.input, test.expected)
	}
}
