package upgrade

import (
	"knative.dev/pkg/test/upgrade"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/e2e"
)

func VerifySugarControllerDeletion(ctx *test.Context) upgrade.Operation {
	return upgrade.NewOperation("Verify sugar-controller deletion", func(c upgrade.Context) {
		if err := test.CheckNoDeployment(ctx.Clients.Kube, "knative-eventing", "sugar-controller"); err != nil {
			c.T.Error(err)
		}
	})
}

func VerifyEventingDashboards(ctx *test.Context) upgrade.Operation {
	return upgrade.NewOperation("Verify eventing dashboards", func(c upgrade.Context) {
		e2e.VerifyDashboards(c.T, ctx, e2e.EventingDashboards)
	})
}
