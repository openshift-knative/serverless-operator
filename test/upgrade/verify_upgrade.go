package upgrade

import (
	"knative.dev/pkg/test/upgrade"

	"github.com/openshift-knative/serverless-operator/test"
)

func VerifySugarControllerDeletion(ctx *test.Context) upgrade.Operation {
	return upgrade.NewOperation("Verify sugar-controller deletion", func(c upgrade.Context) {
		if err := test.CheckNoDeployment(ctx.Clients.Kube, "knative-eventing", "sugar-controller"); err != nil {
			c.T.Error(err)
		}
	})
}
