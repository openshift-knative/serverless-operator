package serving

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestOverrideKourierNamespace(t *testing.T) {
	kourierLabels := map[string]string{
		providerLabel: "kourier",
	}

	withKourier := &unstructured.Unstructured{}
	withKourier.SetNamespace("foo")
	withKourier.SetLabels(kourierLabels)
	withKourier.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "Foo",
		Name:       "bar",
	}})

	want := withKourier.DeepCopy()
	want.SetNamespace("knative-serving-ingress")
	want.SetOwnerReferences(nil)

	overrideKourierNamespace("knative-serving-ingress")(withKourier)

	if !cmp.Equal(withKourier, want) {
		t.Errorf("Resource was not as expected:\n%s", cmp.Diff(withKourier, want))
	}
}

func TestOverrideKourierNamespaceOther(t *testing.T) {
	otherLabels := map[string]string{
		providerLabel: "foo",
	}

	other := &unstructured.Unstructured{}
	other.SetNamespace("foo")
	other.SetLabels(otherLabels)
	other.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "Foo",
		Name:       "bar",
	}})
	want := other.DeepCopy()

	overrideKourierNamespace("knative-serving-ingress")(other)

	if !cmp.Equal(other, want) {
		t.Errorf("Resource was not as expected:\n%s", cmp.Diff(other, want))
	}
}
