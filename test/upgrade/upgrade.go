package upgrade

import (
	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/upgrade/installation"
	pkgupgrade "knative.dev/pkg/test/upgrade"
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

func ServerlessDowngradeOperations(ctx *test.Context) []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		pkgupgrade.NewOperation("DowngradeServerless", func(c pkgupgrade.Context) {
			if err := installation.DowngradeServerless(ctx); err != nil {
				c.T.Error("Serverless downgrade failed:", err)
			}
		}),
	}
}
