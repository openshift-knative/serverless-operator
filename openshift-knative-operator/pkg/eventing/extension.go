package eventing

import (
	"context"
	"os"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"k8s.io/apimachinery/pkg/api/resource"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	operator "knative.dev/operator/pkg/reconciler/common"
)

// NewExtension creates a new extension for a Knative Eventing controller.
func NewExtension(ctx context.Context) operator.Extension {
	return &extension{}
}

type extension struct{}

func (e *extension) Manifests(v1alpha1.KComponent) ([]mf.Manifest, error) {
	return nil, nil
}

func (e *extension) Transformers(v1alpha1.KComponent) []mf.Transformer {
	return nil
}

func (e *extension) Reconcile(ctx context.Context, comp v1alpha1.KComponent) error {
	ke := comp.(*v1alpha1.KnativeEventing)

	// Override images.
	// TODO(SRVCOM-1069): Rethink overriding behavior and/or error surfacing.
	images := common.ImageMapFromEnvironment(os.Environ())
	ke.Spec.Registry.Override = images
	ke.Spec.Registry.Default = images["default"]

	// Ensure webhook has 1G of memory.
	common.EnsureContainerMemoryLimit(&ke.Spec.CommonSpec, "eventing-webhook", resource.MustParse("1024Mi"))

	// SRVKE-500: Ensure we set the SinkBindingSelectionMode to inclusion
	if ke.Spec.SinkBindingSelectionMode == "" {
		ke.Spec.SinkBindingSelectionMode = "inclusion"
	}

	return nil
}

func (e *extension) Finalize(context.Context, v1alpha1.KComponent) error {
	return nil
}
