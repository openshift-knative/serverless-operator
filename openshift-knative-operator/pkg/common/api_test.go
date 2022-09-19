package common

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/ptr"
)

type testVersioner struct {
	version string
	err     error
}

func (t *testVersioner) ServerVersion() (*version.Info, error) {
	return &version.Info{GitVersion: t.version}, t.err
}

func TestVersionCheck(t *testing.T) {
	tests := []struct {
		name          string
		actualVersion *testVersioner
		wantError     bool
	}{{
		name:          "greater version (patch)",
		actualVersion: &testVersioner{version: "v1.20.0"},
	}, {
		name:          "greater version (patch), no v",
		actualVersion: &testVersioner{version: "1.20.0"},
	}, {
		name:          "greater version (patch), pre-release",
		actualVersion: &testVersioner{version: "1.20.2-kpn-065dce"},
	}, {
		name:          "greater version (patch), pre-release with build",
		actualVersion: &testVersioner{version: "1.20.0-1095+9689d22dc3121e-dirty"},
	}, {
		name:          "greater version (minor)",
		actualVersion: &testVersioner{version: "v1.20.0"},
	}, {
		name:          "same version",
		actualVersion: &testVersioner{version: "v1.20.0"},
	}, {
		name:          "same version with build",
		actualVersion: &testVersioner{version: "v1.20.0+k3s.1"},
	}, {
		name:          "same version with pre-release",
		actualVersion: &testVersioner{version: "v1.20.0-k3s.1"},
	}, {
		name:          "smaller version",
		actualVersion: &testVersioner{version: "v1.19.3"},
		wantError:     true,
	}, {
		name:          "error while fetching",
		actualVersion: &testVersioner{err: errors.New("random error")},
		wantError:     true,
	}, {
		name:          "unparseable actual version",
		actualVersion: &testVersioner{version: "v1.19.foo"},
		wantError:     true,
	}}

	minVersion := "1.20.0"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CheckMinimumVersion(test.actualVersion, minVersion)
			if err == nil && test.wantError {
				t.Errorf("Expected an error for minimum: %q, actual: %v", minVersion, test.actualVersion)
			}

			if err != nil && !test.wantError {
				t.Errorf("Expected no error but got %v for minimum: %q, actual: %v", err, minVersion, test.actualVersion)
			}
		})
	}
}

func TestPodSecurityContext(t *testing.T) {
	spec := func(containers ...corev1.Container) appsv1.DeploymentSpec {
		return appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		}
	}

	tests := []struct {
		name       string
		in         *appsv1.Deployment
		deployment string
		container  string
		envs       []corev1.EnvVar
		want       *appsv1.Deployment
	}{{
		name:       "ignore",
		deployment: "foo",
		container:  "container1",
		in: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
			}),
		},
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: ptr.Bool(false),
					ReadOnlyRootFilesystem:   ptr.Bool(true),
					RunAsNonRoot:             ptr.Bool(true),
					Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
					SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
				},
			}),
		},
	}, {
		name:       "ignore",
		deployment: "foo",
		container:  "container1",
		in: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				SecurityContext: &corev1.SecurityContext{
					ReadOnlyRootFilesystem: ptr.Bool(false),
				},
			}),
		},
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: ptr.Bool(false),
					ReadOnlyRootFilesystem:   ptr.Bool(false),
					RunAsNonRoot:             ptr.Bool(true),
					Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
					SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
				},
			}),
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u := &unstructured.Unstructured{}
			if err := scheme.Scheme.Convert(test.in, u, nil); err != nil {
				t.Fatal("Failed to convert deployment to unstructured", err)
			}

			if err := SetSecurityContextForAdmissionController()(u); err != nil {
				t.Fatal("Unexpected error from transformer", err)
			}

			got := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, got, nil); err != nil {
				t.Fatal("Failed to convert unstructured to deployment", err)
			}

			if !cmp.Equal(got, test.want) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", got, test.want, cmp.Diff(got, test.want))
			}
		})
	}
}
