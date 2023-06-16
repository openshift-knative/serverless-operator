package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/base"
)

func TestConfigureEnvValueIfUnset(t *testing.T) {
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
		want       *appsv1.Deployment
	}{{
		name:       "add",
		deployment: "test",
		container:  "container1",
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
				Env:  []corev1.EnvVar{envVar("1", "1"), envVar("foo", "bar")},
			}, corev1.Container{
				Name: "container2",
				Env:  []corev1.EnvVar{envVar("2", "2")},
			}),
		},
	}, {
		name:       "update",
		deployment: "test",
		container:  "container2",
		in: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				Env:  []corev1.EnvVar{envVar("1", "1")},
			}, corev1.Container{
				Name: "container2",
				Env:  []corev1.EnvVar{envVar("2", "2"), envVar("foo", "to_be_updated")},
			}),
		},
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: spec(corev1.Container{
				Name: "container1",
				Env:  []corev1.EnvVar{envVar("1", "1")},
			}, corev1.Container{
				Name: "container2",
				Env:  []corev1.EnvVar{envVar("2", "2"), envVar("foo", "bar")},
			}),
		},
	}}

	s := &base.CommonSpec{Workloads: []base.WorkloadOverride{{
		Name: "net-kourier-controller",
		Env: []base.EnvRequirementsOverride{{
			Container: "controller",
			EnvVars: []corev1.EnvVar{{
				Name:  "KUBE_API_BURST",
				Value: "100",
			}},
		}},
	}}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u := &unstructured.Unstructured{}
			if err := scheme.Scheme.Convert(test.in, u, nil); err != nil {
				t.Fatal("Failed to convert deployment to unstructured", err)
			}

			if err := ConfigureEnvValueIfUnset(s, test.deployment, test.container, "foo", "bar")(u); err != nil {
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
