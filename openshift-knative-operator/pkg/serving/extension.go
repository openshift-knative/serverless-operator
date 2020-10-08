package serving

import (
	"context"

	mf "github.com/manifestival/manifestival"
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
	return nil
}

func (e *extension) Finalize(context.Context, v1alpha1.KComponent) error {
	return nil
}
