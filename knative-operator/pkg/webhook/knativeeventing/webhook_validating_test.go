package knativeeventing

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	ke1 = &operatorv1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ke1",
		},
	}
	ke2 = &operatorv1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ke2",
		},
	}

	decoder *admission.Decoder
)

func init() {
	apis.AddToScheme(scheme.Scheme)
	decoder = admission.NewDecoder(scheme.Scheme)
}

func TestInvalidNamespace(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_EVENTING_NAMESPACE", "knative-eventing")

	validator := NewValidator(nil, decoder)

	req, err := testutil.RequestFor(ke1)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", ke1, err)
	}

	result := validator.Handle(context.Background(), req)
	if result.Allowed {
		t.Error("The required namespace is wrong, but the request is allowed")
	}
}

func TestLoneliness(t *testing.T) {
	os.Clearenv()

	validator := NewValidator(fake.NewClientBuilder().WithObjects(ke2).Build(), decoder)

	req, err := testutil.RequestFor(ke1)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", ke1, err)
	}

	result := validator.Handle(context.Background(), req)
	if result.Allowed {
		t.Errorf("Too many KnativeEventings: %v", result.AdmissionResponse)
	}
}
