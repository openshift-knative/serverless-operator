package knativeserving_test

import (
	"context"
	"os"
	"testing"

	. "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeserving"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
}

func TestInvalidNamespace(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_SERVING_NAMESPACE", "knative-serving")
	validator := KnativeServingValidator{}
	// The mock will return a KS in the 'default' namespace
	validator.InjectDecoder(&mockDecoder{})
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The required namespace is wrong, but the request is allowed")
	}
}

func TestInvalidMajorVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.1.13")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{})
	validator.InjectClient(fake.NewFakeClient(mockClusterVersion("3.2.0")))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The version is too low, but the request is allowed")
	}
}

func TestInvalidMinorVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.1.13")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{})
	validator.InjectClient(fake.NewFakeClient(mockClusterVersion("4.0.0")))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The version is too low, but the request is allowed")
	}
}

func TestInvalidPatchVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.1.13")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{})
	validator.InjectClient(fake.NewFakeClient(mockClusterVersion("4.1.12")))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The version is too low, but the request is allowed")
	}
}

func TestInvalidPrereleaseVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.3.13")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{})
	validator.InjectClient(fake.NewFakeClient(mockClusterVersion("4.1.13-alpha")))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The version is too low, but the request is allowed")
	}
}

func TestValidMajorVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.3.0-0")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{})
	validator.InjectClient(fake.NewFakeClient(mockClusterVersion("4.3.0")))
	result := validator.Handle(context.TODO(), types.Request{})
	if !result.Response.Allowed {
		t.Error("The version is later, but the request is not allowed")
	}
}

func TestValidMinorVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.3.0-0")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{})
	validator.InjectClient(fake.NewFakeClient(mockClusterVersion("4.4.0")))
	result := validator.Handle(context.TODO(), types.Request{})
	if !result.Response.Allowed {
		t.Error("The version is later, but the request is not allowed")
	}
}

func TestValidPatchVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.3.0-0")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{})
	validator.InjectClient(fake.NewFakeClient(mockClusterVersion("4.3.1")))
	result := validator.Handle(context.TODO(), types.Request{})
	if !result.Response.Allowed {
		t.Error("The version is later, but the request is not allowed")
	}
}

func TestValidPrereleaseAlphaVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.3.0-0")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{})
	validator.InjectClient(fake.NewFakeClient(mockClusterVersion("4.3.0-alpha")))
	result := validator.Handle(context.TODO(), types.Request{})
	if !result.Response.Allowed {
		t.Error("The version is later, but the request is not allowed")
	}
}

func TestValidPrereleaseNightlyVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.3.0-0")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{})
	validator.InjectClient(fake.NewFakeClient(mockClusterVersion("4.3.0-0.nightly-2020-01-28-050934")))
	result := validator.Handle(context.TODO(), types.Request{})
	if !result.Response.Allowed {
		t.Error("The version is later, but the request is not allowed")
	}
}

func mockClusterVersion(version string) *configv1.ClusterVersion {
	return &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			Desired: configv1.Update{
				Version: version,
			},
		},
	}
}

type mockDecoder struct {
	ks *servingv1alpha1.KnativeServing
}

var _ types.Decoder = (*mockDecoder)(nil)

func (mock *mockDecoder) Decode(_ types.Request, obj runtime.Object) error {
	if p, ok := obj.(*servingv1alpha1.KnativeServing); ok {
		if mock.ks != nil {
			*p = *mock.ks
		} else {
			*p = servingv1alpha1.KnativeServing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "knative-serving",
					Namespace: "default",
				},
			}
		}
	}
	return nil
}
