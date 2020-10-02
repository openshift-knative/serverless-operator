package eventing

import (
	"context"
	"os"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/new-operator/pkg/common"
	"k8s.io/apimachinery/pkg/api/resource"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	operator "knative.dev/operator/pkg/reconciler/common"
)

func NewExtension(ctx context.Context) operator.Extension {
	return &extension{}
}

type extension struct{}

func (e *extension) Transformers(v1alpha1.KComponent) []mf.Transformer {
	return nil
}

func (e *extension) Reconcile(ctx context.Context, comp v1alpha1.KComponent) error {
	ks := comp.(*v1alpha1.KnativeEventing)

	configureImagesFromEnvironment(ks)
	common.EnsureContainerMemoryLimit(&ks.Spec.CommonSpec, "eventing-webhook", resource.MustParse("1024Mi"))
	return nil
}

func (e *extension) Finalize(context.Context, v1alpha1.KComponent) error {
	return nil
}

func configureImagesFromEnvironment(ks *v1alpha1.KnativeEventing) {
	reg := ks.GetSpec().GetRegistry()

	reg.Override = common.ImageMapFromEnvironment(os.Environ())

	if defaultVal, ok := reg.Override["default"]; ok {
		reg.Default = defaultVal
	}
}
