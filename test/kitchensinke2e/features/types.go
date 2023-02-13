package features

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
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

type component interface {
	installable
	kreferencable
	readyable
	labelable
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
