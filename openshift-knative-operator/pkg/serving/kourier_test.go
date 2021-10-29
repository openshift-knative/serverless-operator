package serving

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
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

	ks := &operatorv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "knative-serving",
			Name:      "test",
		},
	}

	want := withKourier.DeepCopy()
	want.SetNamespace("knative-serving-ingress")
	want.SetLabels(map[string]string{
		providerLabel:                  "kourier",
		socommon.ServingOwnerNamespace: ks.Namespace,
		socommon.ServingOwnerName:      ks.Name,
	})
	want.SetOwnerReferences(nil)

	overrideKourierNamespace(ks)(withKourier)

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

	ks := &operatorv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "knative-serving",
			Name:      "test",
		},
	}

	overrideKourierNamespace(ks)(other)

	if !cmp.Equal(other, want) {
		t.Errorf("Resource was not as expected:\n%s", cmp.Diff(other, want))
	}
}
