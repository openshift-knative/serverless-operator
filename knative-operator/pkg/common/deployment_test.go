package common_test

import (
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/ptr"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	deploymentName = "controller"
	proxyValue     = "http://192.168.130.11:30001"
	namespace      = "default"
	servingName    = "knative-serving"
	noProxy        = "index.docker.io"
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

func TestProxySettingWithInvalidController(t *testing.T) {
	client := fake.NewFakeClient(
		mockController(appsv1.DeploymentSpec{}, "invalid"))
	ks := &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      servingName,
			Namespace: namespace,
		},
	}
	if err := common.ApplyProxySettings(ks, client); err != nil {
		t.Error(err)
	}
}

func TestProxySettingForHTTPProxy(t *testing.T) {
	os.Setenv("HTTP_PROXY", proxyValue)
	client := fake.NewFakeClient(
		mockController(appsv1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Env: []v1.EnvVar{{
							Name:  "HTTP_PROXY",
							Value: proxyValue,
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
	if err := common.ApplyProxySettings(ks, client); err != nil {
		t.Error(err)
	}
}

func TestProxySettingForHTTPSProxy(t *testing.T) {
	os.Setenv("HTTPS_PROXY", proxyValue)
	client := fake.NewFakeClient(
		mockController(appsv1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Env: []v1.EnvVar{{
							Name:  "HTTPS_PROXY",
							Value: proxyValue,
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
	if err := common.ApplyProxySettings(ks, client); err != nil {
		t.Error(err)
	}
}

func TestProxySettingForNonExistedKey(t *testing.T) {
	os.Setenv("NO_PROXY", noProxy)
	client := fake.NewFakeClient(
		mockController(appsv1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Env: []v1.EnvVar{{
							Name:  "HTTP_PROXY",
							Value: proxyValue,
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
	if err := common.ApplyProxySettings(ks, client); err != nil {
		t.Error(err)
	}
}

func TestProxySettingWithSameKeyEmptyValue(t *testing.T) {
	os.Setenv("HTTP_PROXY", "")
	client := fake.NewFakeClient(
		mockController(appsv1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Env: []v1.EnvVar{{
							Name:  "HTTP_PROXY",
							Value: proxyValue,
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
	if err := common.ApplyProxySettings(ks, client); err != nil {
		t.Error(err)
	}
}
