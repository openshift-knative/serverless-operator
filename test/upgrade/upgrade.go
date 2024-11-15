package upgrade

import (
	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/upgrade/installation"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	"knative.dev/reconciler-test/pkg/environment"
)

func ServerlessUpgradeOperations(ctx *test.Context) []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		pkgupgrade.NewOperation("UpgradeServerless", func(c pkgupgrade.Context) {
			if err := installation.UpgradeServerless(ctx); err != nil {
				c.T.Error("Serverless upgrade failed:", err)
			}
		}),
	}
}

func ServerlessDowngradeOperations(ctx *test.Context, glob environment.GlobalEnvironment) []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		pkgupgrade.NewOperation("DowngradeServerless", func(c pkgupgrade.Context) {
			if err := installation.DowngradeServerless(ctx); err != nil {
				c.T.Error("Serverless downgrade failed:", err)
			}
			// Tests are disabled on z-stream releases. It was relevant for 1.33 -> 1.34 upgrade/downgrade scenario
			//installation.CleanupTriggerv2ConsumerGroups(c, glob)
			//installation.CleanupTriggerv2Deployments(c, glob)
		}),
	}
}
