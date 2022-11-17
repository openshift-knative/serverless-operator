package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

// TestInjectEnvironment tests InjectEnvironmentIntoDeployment and InjectCommonEnvironment.
func TestInjectEnvironment(t *testing.T) {
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
		commonEnvs []corev1.EnvVar
		want       *appsv1.Deployment
	}{{
		name:       "ignore",
		deployment: "foo",
		container:  "container1",
		envs:       []corev1.EnvVar{envVar("foo", "bar")},
		commonEnvs: []corev1.EnvVar{envVar("KUBERNETES_MIN_VERSION", "v1.0.0")},
		in: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				Env:  []corev1.EnvVar{envVar("1", "1")},
			}),
		},
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				Env:  []corev1.EnvVar{envVar("1", "1"), envVar("KUBERNETES_MIN_VERSION", "v1.0.0")},
			}),
		},
	}, {
		name:       "append",
		deployment: "test",
		container:  "container1",
		envs:       []corev1.EnvVar{envVar("foo", "bar")},
		commonEnvs: []corev1.EnvVar{envVar("KUBERNETES_MIN_VERSION", "v1.0.0")},
		in: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				Env:  []corev1.EnvVar{envVar("1", "1")},
			}, corev1.Container{
				Name: "container2",
				Env:  []corev1.EnvVar{envVar("2", "2")},
			}),
		},
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				Env:  []corev1.EnvVar{envVar("1", "1"), envVar("foo", "bar"), envVar("KUBERNETES_MIN_VERSION", "v1.0.0")},
			}, corev1.Container{
				Name: "container2",
				Env:  []corev1.EnvVar{envVar("2", "2"), envVar("KUBERNETES_MIN_VERSION", "v1.0.0")},
			}),
		},
	}, {
		name:       "update",
		deployment: "test",
		container:  "container2",
		envs:       []corev1.EnvVar{envVar("2", "bar")},
		commonEnvs: []corev1.EnvVar{envVar("KUBERNETES_MIN_VERSION", "v1.0.0")},
		in: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				Env:  []corev1.EnvVar{envVar("1", "1")},
			}, corev1.Container{
				Name: "container2",
				Env:  []corev1.EnvVar{envVar("2", "2")},
			}),
		},
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				Env:  []corev1.EnvVar{envVar("1", "1"), envVar("KUBERNETES_MIN_VERSION", "v1.0.0")},
			}, corev1.Container{
				Name: "container2",
				Env:  []corev1.EnvVar{envVar("2", "bar"), envVar("KUBERNETES_MIN_VERSION", "v1.0.0")},
			}),
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u := &unstructured.Unstructured{}
			if err := scheme.Scheme.Convert(test.in, u, nil); err != nil {
				t.Fatal("Failed to convert deployment to unstructured", err)
			}

			if err := InjectEnvironmentIntoDeployment(test.deployment, test.container, test.envs...)(u); err != nil {
				t.Fatal("Unexpected error from transformer", err)
			}

			if err := InjectCommonEnvironment(test.commonEnvs...)(u); err != nil {
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

func TestUpsert(t *testing.T) {
	tests := []struct {
		name string
		in   []corev1.EnvVar
		add  corev1.EnvVar
		want []corev1.EnvVar
	}{{
		name: "nil",
		in:   nil,
		add:  envVar("foo", "bar"),
		want: []corev1.EnvVar{envVar("foo", "bar")},
	}, {
		name: "empty",
		in:   []corev1.EnvVar{},
		add:  envVar("foo", "bar"),
		want: []corev1.EnvVar{envVar("foo", "bar")},
	}, {
		name: "append",
		in:   []corev1.EnvVar{envVar("foo", "bar")},
		add:  envVar("foo2", "bar2"),
		want: []corev1.EnvVar{envVar("foo", "bar"), envVar("foo2", "bar2")},
	}, {
		name: "update",
		in:   []corev1.EnvVar{envVar("foo", "bar")},
		add:  envVar("foo", "baz"),
		want: []corev1.EnvVar{envVar("foo", "baz")},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := upsert(test.in, test.add)
			if !cmp.Equal(got, test.want) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", got, test.want, cmp.Diff(got, test.want))
			}
		})
	}
}

func envVar(name, value string) corev1.EnvVar {
	return corev1.EnvVar{Name: name, Value: value}
}
