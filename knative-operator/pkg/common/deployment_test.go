package common_test

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	deploymentName = "controller"
	namespace      = "default"
)

func deployment(name string, containers ...v1.Container) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: containers,
				},
			},
		},
	}
}

func container(env ...v1.EnvVar) v1.Container {
	return v1.Container{
		Env: env,
	}
}

func envVar(name, value string) v1.EnvVar {
	return v1.EnvVar{Name: name, Value: value}
}

func TestApplyEnvironmentToDeployment(t *testing.T) {
	tests := []struct {
		name   string
		in     *appsv1.Deployment
		env    map[string]string
		expect *appsv1.Deployment
	}{{
		name:   "not found",
		in:     deployment("something else"),
		env:    map[string]string{"test": "foo"},
		expect: deployment("something else"),
	}, {
		name:   "add vars",
		in:     deployment(deploymentName, container(envVar("other", "bar"))),
		env:    map[string]string{"test": "foo", "test2": "foo2"},
		expect: deployment(deploymentName, container(envVar("other", "bar"), envVar("test", "foo"), envVar("test2", "foo2"))),
	}, {
		name:   "change var",
		in:     deployment(deploymentName, container(envVar("test", "bar"))),
		env:    map[string]string{"test": "foo"},
		expect: deployment(deploymentName, container(envVar("test", "foo"))),
	}, {
		name:   "delete var",
		in:     deployment(deploymentName, container(envVar("test", "bar"))),
		env:    map[string]string{"test": ""},
		expect: deployment(deploymentName, container()),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithObjects(test.in).Build()
			if err := common.ApplyEnvironmentToDeployment(namespace, deploymentName, test.env, c); err != nil {
				t.Fatalf("ApplyEnvironmentToDeployment = %v, want no error", err)
			}

			if test.expect != nil {
				got := &appsv1.Deployment{}
				if err := c.Get(context.TODO(), client.ObjectKey{Name: test.expect.Name, Namespace: test.expect.Namespace}, got); err != nil {
					t.Fatalf("Deployment.Get = %v, want no error", err)
				}

				sortEnv(got)
				sortEnv(test.expect)

				// Unset as the new fake clients touch this.
				got.ResourceVersion = ""

				if !cmp.Equal(test.expect, got) {
					t.Errorf("Deployment not as expected, diff: %s", cmp.Diff(got, test.expect))
				}
			}
		})
	}
}

func sortEnv(deploy *appsv1.Deployment) {
	for i := range deploy.Spec.Template.Spec.Containers {
		container := &deploy.Spec.Template.Spec.Containers[i]
		sort.Slice(container.Env, func(i, j int) bool {
			return container.Env[i].Name < container.Env[j].Name
		})
	}
}
