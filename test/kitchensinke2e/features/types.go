package features

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/manifest"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/reconciler-test/pkg/feature"
)

type installable interface {
	Install(name string, opts ...manifest.CfgFn) feature.StepFn
}

type kreferencable interface {
	KReference(name string) *duckv1.KReference
}

type readyable interface {
	IsReady(name string, timings ...time.Duration) feature.StepFn
}

type labelable interface {
	ShortLabel() string
	Label() string
}

type deletable interface {
	Delete(name string) feature.StepFn
}

type component interface {
	installable
	kreferencable
	readyable
	labelable
	deletable
}

type genericComponent struct {
	kind    string
	gvr     schema.GroupVersionResource
	install func(name string, opts ...manifest.CfgFn) feature.StepFn
	isReady func(name string, timings ...time.Duration) feature.StepFn

	// shortLabel is used as a prefix in resource names, so must meet requirements for k8s names
	shortLabel string

	// label is used in test descriptions
	label string
}

func (c genericComponent) ShortLabel() string {
	return c.shortLabel
}

func (c genericComponent) Label() string {
	return c.label
}

func (c genericComponent) KReference(name string) *duckv1.KReference {
	return &duckv1.KReference{
		Kind:       c.kind,
		Name:       name,
		APIVersion: c.gvr.GroupVersion().String(),
	}
}

func (c genericComponent) IsReady(name string, timings ...time.Duration) feature.StepFn {
	if c.isReady != nil {
		return c.isReady(name, timings...)
	}

	return k8s.IsReady(c.gvr, name, timings...)
}

func (c genericComponent) Install(name string, opts ...manifest.CfgFn) feature.StepFn {
	return c.install(name, opts...)
}

func shortLabel(thing labelable) string {
	if thing != nil {
		return thing.ShortLabel()
	}
	return "n"
}

func label(thing labelable) string {
	if thing != nil {
		return thing.Label()
	}
	return "nil"
}

func (c genericComponent) Delete(name string) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		dc := dynamicclient.Get(ctx)
		ref := c.KReference(name)

		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			t.Errorf("Could not parse GroupVersion for %+v", ref.APIVersion)
		}

		resource := apis.KindToResource(gv.WithKind(ref.Kind))
		t.Logf("Deleting %s/%s of GVR: %+v", ref.Namespace, ref.Name, resource)

		deleteOptions := &metav1.DeleteOptions{}
		// Set delete propagation policy to foreground
		foregroundDeletePropagation := metav1.DeletePropagationForeground
		deleteOptions.PropagationPolicy = &foregroundDeletePropagation

		err = dc.Resource(resource).Namespace(ref.Namespace).Delete(ctx, ref.Name, *deleteOptions)
		// Ignore not found errors.
		if err != nil && !apierrors.IsNotFound(err) {
			t.Logf("Warning, failed to delete %s/%s of GVR: %+v: %v", ref.Namespace, ref.Name, resource, err)
		}
	}
}
