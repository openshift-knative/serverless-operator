package common_test

import (
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/ptr"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	deploymentName = "controller"
	httpProxy      = "http://192.168.130.11:30001"
	noProxy        = "index.docker.io"
	namespace      = "default"
	servingName    = "knative-serving"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
}

func mockController(spec appsv1.DeploymentSpec, name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
	}
}

func TestUpdateWithInvalidController(t *testing.T) {
	client := fake.NewFakeClient()
	ks := &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      servingName,
			Namespace: namespace,
		},
	}
	if err := common.ApplyProxySettings(ks, client); err != nil && apierrors.IsNotFound(err) {
		t.Error(err)
	}
}

func TestUpdateWithPodSpec(t *testing.T) {
	os.Setenv("HTTP_PROXY", httpProxy)
	client := fake.NewFakeClient(
		mockController(appsv1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Env: []v1.EnvVar{{
							Name:  "CONFIG_LOGGING_NAME",
							Value: "config-logging",
						}, {
							Name:  "NO_PROXY",
							Value: noProxy,
						}},
					}},
				},
			},
		}, deploymentName))
	ks := &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      servingName,
			Namespace: namespace,
		},
	}
	if err := common.ApplyProxySettings(ks, client); err != nil && apierrors.IsNotFound(err) {
		t.Error(err)
	}
}

func TestUpdateWithPodSpecWithSameKey(t *testing.T) {
	os.Setenv("HTTP_PROXY", httpProxy)
	client := fake.NewFakeClient(
		mockController(appsv1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Env: []v1.EnvVar{{
							Name:  "CONFIG_LOGGING_NAME",
							Value: "config-logging",
						}, {
							Name:  "HTTP_PROXY",
							Value: httpProxy,
						}},
					}},
				},
			},
		}, deploymentName))
	ks := &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      servingName,
			Namespace: namespace,
		},
	}
	if err := common.ApplyProxySettings(ks, client); err != nil && apierrors.IsNotFound(err) {
		t.Error(err)
	}
}

func TestUpdateWithPodSpecWithSameKeyEmptyValue(t *testing.T) {
	os.Setenv("HTTP_PROXY", "")
	client := fake.NewFakeClient(
		mockController(appsv1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Env: []v1.EnvVar{{
							Name:  "CONFIG_LOGGING_NAME",
							Value: "config-logging",
						}, {
							Name:  "HTTP_PROXY",
							Value: httpProxy,
						}},
					}},
				},
			},
		}, deploymentName))
	ks := &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      servingName,
			Namespace: namespace,
		},
	}
	if err := common.ApplyProxySettings(ks, client); err != nil && apierrors.IsNotFound(err) {
		t.Error(err)
	}
}
