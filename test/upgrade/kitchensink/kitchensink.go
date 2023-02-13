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
)

type FeatureUpgradeTest struct {
	Context     context.Context
	Environment environment.Environment
	Feature     *feature.Feature
}

func NewFeatureUpgradeTest(t *testing.T, global environment.GlobalEnvironment, f *feature.Feature) FeatureUpgradeTest {
	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(4*time.Second, 600*time.Second),
		environment.Managed(t),
	)
	return FeatureUpgradeTest{
		Context:     ctx,
		Environment: env,
		Feature:     f,
	}
}

func (f FeatureUpgradeTest) PreUpgrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation(f.Feature.Name+"PreUpgrade", func(c pkgupgrade.Context) {
		setups := filterStepTimings(f.Feature.Steps, feature.Setup)
		for _, s := range setups {
			//s.Fn()
		}
		requirements := filterStepTimings(f.Feature.Steps, feature.Requirement)
		for _, r := range requirements {
			//r.Fn()
		}
	})
}

func (f FeatureUpgradeTest) PostUpgrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation(f.Feature.Name+"PostUpgrade", func(c pkgupgrade.Context) {
		asserts := filterStepTimings(f.Feature.Steps, feature.Assert)
		for _, a := range asserts {
			//s.Fn()
		}
	})
}

func (f FeatureUpgradeTest) PostDowngrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation(f.Feature.Name+"PostUpgrade", func(c pkgupgrade.Context) {
		asserts := filterStepTimings(f.Feature.Steps, feature.Assert)
		for _, a := range asserts {
			//s.Fn()
		}
		teardowns := filterStepTimings(f.Feature.Steps, feature.Teardown)
		for _, td := range teardowns {
			//td.Fn()
		}
	})
}

type FeatureTestGroup []FeatureUpgradeTest

func (fg FeatureTestGroup) PreUpgradeTests() []pkgupgrade.Operation {
	var ops []pkgupgrade.Operation
	for _, ft := range fg {
		ops = append(ops, ft.PreUpgrade())
	}
	return ops
}

func (fg FeatureTestGroup) PostUpgradeTests() []pkgupgrade.Operation {
	var ops []pkgupgrade.Operation
	for _, ft := range fg {
		ops = append(ops, ft.PostUpgrade())
	}
	return ops
}

func (fg FeatureTestGroup) PostDowngradeTests() []pkgupgrade.Operation {
	var ops []pkgupgrade.Operation
	for _, ft := range fg {
		ops = append(ops, ft.PostDowngrade())
	}
	return ops
}

//func categorizeSteps(steps []feature.Step) map[feature.Timing][]feature.Step {
//	res := make(map[feature.Timing][]feature.Step, 4)
//
//	res[feature.Setup] = filterStepTimings(steps, feature.Setup)
//	res[feature.Requirement] = filterStepTimings(steps, feature.Requirement)
//	res[feature.Assert] = filterStepTimings(steps, feature.Assert)
//	res[feature.Teardown] = filterStepTimings(steps, feature.Teardown)
//
//	return res
//}

func filterStepTimings(steps []feature.Step, timing feature.Timing) []feature.Step {
	var res []feature.Step
	for _, s := range steps {
		if s.T == timing {
			res = append(res, s)
		}
	}
	return res
}
