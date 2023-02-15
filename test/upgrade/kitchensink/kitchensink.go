package kitchensink

import (
	"context"
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
	Context     *context.Context
	Global      environment.GlobalEnvironment
	Environment *environment.Environment
	Feature     *feature.Feature
}

func (fe *FeatureWithEnvironment) CreateEnvironment() {
	ctx, env := fe.Global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(4*time.Second, 600*time.Second),
		//environment.Managed(t),
	)

	// Copied from reconciler-test/MagicEnvironment.
	// The Store that is inside of the Feature will be assigned to the context.
	// If no Store is set on Feature, Test will create a new store.KVStore
	// and set it on the feature and then apply it to the Context.
	if fe.Feature.State == nil {
		fe.Feature.State = &state.KVStore{}
	}
	ctx = state.ContextWith(ctx, fe.Feature.State)
	ctx = feature.ContextWith(ctx, fe.Feature)

	fe.Context = &ctx
	fe.Environment = &env
}

func (fe *FeatureWithEnvironment) PreUpgrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation(fe.Feature.Name, func(c pkgupgrade.Context) {
		c.T.Parallel()
		fe.CreateEnvironment()
		setups := filterStepTimings(fe.Feature.Steps, feature.Setup)
		for _, s := range setups {
			s.Fn(*fe.Context, c.T)
		}
		requirements := filterStepTimings(fe.Feature.Steps, feature.Requirement)
		for _, r := range requirements {
			r.Fn(*fe.Context, c.T)
		}
		asserts := filterStepTimings(fe.Feature.Steps, feature.Assert)
		for _, a := range asserts {
			a.Fn(*fe.Context, c.T)
		}
	})
}

func (fe *FeatureWithEnvironment) PostUpgrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation(fe.Feature.Name, func(c pkgupgrade.Context) {
		c.T.Parallel()
		requirements := filterStepTimings(fe.Feature.Steps, feature.Requirement)
		for _, r := range requirements {
			r.Fn(*fe.Context, c.T)
		}
		asserts := filterStepTimings(fe.Feature.Steps, feature.Assert)
		for _, a := range asserts {
			a.Fn(*fe.Context, c.T)
		}
		teardowns := filterStepTimings(fe.Feature.Steps, feature.Teardown)
		for _, td := range teardowns {
			td.Fn(*fe.Context, c.T)
		}
	})
}

type FeatureWithEnvironmentGroup []*FeatureWithEnvironment

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
