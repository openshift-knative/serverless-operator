package knativekafka_test

import (
	"context"
	"os"
	"testing"

	operatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	. "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativekafka"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/testutil"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
	operatorv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
}

var ke1 = &operatorv1alpha1.KnativeKafka{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ke1",
	},
	Spec: operatorv1alpha1.KnativeKafkaSpec{
		Source: operatorv1alpha1.Source{
			Enabled: false,
		},
		Channel: operatorv1alpha1.Channel{
			Enabled: false,
		},
	},
}
var ke2 = &operatorv1alpha1.KnativeKafka{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ke2",
	},
	Spec: operatorv1alpha1.KnativeKafkaSpec{
		Source: operatorv1alpha1.Source{
			Enabled: false,
		},
		Channel: operatorv1alpha1.Channel{
			Enabled: false,
		},
	},
}
var ke3 = &operatorv1alpha1.KnativeKafka{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ke3",
	},
	Spec: operatorv1alpha1.KnativeKafkaSpec{
		Source: operatorv1alpha1.Source{
			Enabled: false,
		},
		Channel: operatorv1alpha1.Channel{
			Enabled: true,
		},
	},
}

func TestInvalidNamespace(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")
	validator := KnativeKafkaValidator{}
	validator.InjectDecoder(&mockDecoder{ke1})
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The required namespace is wrong, but the request is allowed")
	}
}

func TestInvalidVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.1.13")
	validator := KnativeKafkaValidator{}
	validator.InjectDecoder(&mockDecoder{ke1})
	validator.InjectClient(fake.NewFakeClient(testutil.MockClusterVersion("3.2.0")))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The version is too low, but the request is allowed")
	}
}

func TestLoneliness(t *testing.T) {
	os.Clearenv()
	validator := KnativeKafkaValidator{}
	validator.InjectDecoder(&mockDecoder{ke1})
	validator.InjectClient(fake.NewFakeClient(ke2))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Errorf("Too many KnativeKafkas: %v", result.Response)
	}
}

func TestShape(t *testing.T) {
	os.Clearenv()
	validator := KnativeKafkaValidator{}
	validator.InjectDecoder(&mockDecoder{ke3})
	validator.InjectClient(fake.NewFakeClient())
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The shape is invalid, but the request is allowed")
	}
}

type mockDecoder struct {
	ke *operatorv1alpha1.KnativeKafka
}

var _ types.Decoder = (*mockDecoder)(nil)

func (mock *mockDecoder) Decode(_ types.Request, obj runtime.Object) error {
	if p, ok := obj.(*operatorv1alpha1.KnativeKafka); ok {
		*p = *mock.ke
	}
	return nil
}
