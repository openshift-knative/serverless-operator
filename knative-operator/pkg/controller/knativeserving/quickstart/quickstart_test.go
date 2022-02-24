package quickstart

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	apierrs "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	os.Setenv(EnvKey, "../../../../deploy/resources/quickstart/serverless-application-quickstart.yaml")
	apis.AddToScheme(scheme.Scheme)
}

func TestQuickstartErrors(t *testing.T) {
	someErr := errors.New("test")

	tests := []struct {
		err      error
		expected error
	}{{
		err:      nil,
		expected: nil,
	}, {
		err:      someErr,
		expected: someErr,
	}, {
		err:      &apierrs.NoKindMatchError{},
		expected: nil,
	}}

	for _, test := range tests {
		if err := Apply(&fakeClient{err: test.err}); !errors.Is(err, test.expected) {
			t.Errorf("Apply() = %v, want %v", err, test.expected)
		}
		if err := Delete(&fakeClient{err: test.err}); !errors.Is(err, test.expected) {
			t.Errorf("Delete() = %v, want %v", err, test.expected)
		}
	}
}

type fakeClient struct {
	client.Client

	err error
}

func (f *fakeClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return f.err
}

func (f *fakeClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return f.err
}

func (f *fakeClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return f.err
}

func (f *fakeClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return f.err
}
