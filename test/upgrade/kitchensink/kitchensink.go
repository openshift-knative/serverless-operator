package kitchensink

import (
	"context"
	"testing"
	"time"

	"knative.dev/pkg/system"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
	"knative.dev/reconciler-test/pkg/state"
)

type FeatureWithEnvironment struct {
	Context     context.Context
	Environment environment.Environment
	Feature     *feature.Feature
}

func NewFeatureWithEnvironment(t *testing.T, global environment.GlobalEnvironment, f *feature.Feature) FeatureWithEnvironment {
	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(4*time.Second, 600*time.Second),
		environment.Managed(t),
	)

	// Copied from reconciler-test/MagicEnvironment.
	// The Store that is inside of the Feature will be assigned to the context.
	// If no Store is set on Feature, Test will create a new store.KVStore
	// and set it on the feature and then apply it to the Context.
	if f.State == nil {
		f.State = &state.KVStore{}
	}
	ctx = state.ContextWith(ctx, f.State)
	ctx = feature.ContextWith(ctx, f)

	return FeatureWithEnvironment{
		Context:     ctx,
		Environment: env,
		Feature:     f,
	}
}

func (f FeatureWithEnvironment) PreUpgrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation(f.Feature.Name, func(c pkgupgrade.Context) {
		setups := filterStepTimings(f.Feature.Steps, feature.Setup)
		for _, s := range setups {
			s.Fn(f.Context, c.T)
		}
		requirements := filterStepTimings(f.Feature.Steps, feature.Requirement)
		for _, r := range requirements {
			r.Fn(f.Context, c.T)
		}
		asserts := filterStepTimings(f.Feature.Steps, feature.Assert)
		for _, a := range asserts {
			a.Fn(f.Context, c.T)
		}
	})
}

func (f FeatureWithEnvironment) PostUpgrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation(f.Feature.Name, func(c pkgupgrade.Context) {
		requirements := filterStepTimings(f.Feature.Steps, feature.Requirement)
		for _, r := range requirements {
			r.Fn(f.Context, c.T)
		}
		asserts := filterStepTimings(f.Feature.Steps, feature.Assert)
		for _, a := range asserts {
			a.Fn(f.Context, c.T)
		}
		teardowns := filterStepTimings(f.Feature.Steps, feature.Teardown)
		for _, td := range teardowns {
			td.Fn(f.Context, c.T)
		}
	})
}

type FeatureWithEnvironmentGroup []FeatureWithEnvironment

func (fg FeatureWithEnvironmentGroup) PreUpgradeTests() []pkgupgrade.Operation {
	var ops []pkgupgrade.Operation
	for _, ft := range fg {
		ops = append(ops, ft.PreUpgrade())
	}
	return ops
}

func (fg FeatureWithEnvironmentGroup) PostUpgradeTests() []pkgupgrade.Operation {
	var ops []pkgupgrade.Operation
	for _, ft := range fg {
		ops = append(ops, ft.PostUpgrade())
	}
	return ops
}

func filterStepTimings(steps []feature.Step, timing feature.Timing) []feature.Step {
	var res []feature.Step
	for _, s := range steps {
		if s.T == timing {
			res = append(res, s)
		}
	}
	return res
}
