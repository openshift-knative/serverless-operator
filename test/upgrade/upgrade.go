package upgrade

import (
	pkgupgrade "knative.dev/pkg/test/upgrade"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/upgrade/installation"
)

func ServerlessUpgradeOperations(ctx *test.Context) []pkgupgrade.Operation {
	lifecycle := installation.NewServerlessLifecycle(test.Flags.OLMVersion)
	return []pkgupgrade.Operation{
		pkgupgrade.NewOperation("UpgradeServerless", func(c pkgupgrade.Context) {
			if err := lifecycle.Upgrade(ctx); err != nil {
				c.T.Error("Serverless upgrade failed:", err)
			}
		}),
	}
}

func ServerlessDowngradeOperations(ctx *test.Context) []pkgupgrade.Operation {
	lifecycle := installation.NewServerlessLifecycle(test.Flags.OLMVersion)
	return []pkgupgrade.Operation{
		pkgupgrade.NewOperation("DowngradeServerless", func(c pkgupgrade.Context) {
			if err := lifecycle.Downgrade(ctx); err != nil {
				c.T.Error("Serverless downgrade failed:", err)
			}
		}),
	}
}
