package e2e

import (
	"github.com/openshift-knative/serverless-operator/test"
	"os"
	"testing"
)

func createSubscriptionAndWaitForCSVtoSucceed(t *testing.T, ctx *test.Context) {
	t.Run("create subscription and wait for CSV to succeed", func(t *testing.T) {
		_, err := test.WithOperatorReady(ctx, "serverless-operator-subscription")
		if err != nil {
			t.Fatal("Failed", err)
		}
	})
}

func undeployServerlessOperatorAndCheckDependentOperatorsRemoved(t *testing.T, ctx *test.Context) {
	if t.Failed() && runsOnCiOperator() {
		t.Log("Skipping updeployment of serverless as tests failed and we are running on Openshift CI")
		return
	}
	t.Run("undeploy serverless operator and check dependent operators removed", func(t *testing.T) {
		ctx.Cleanup()
		err := test.WaitForOperatorDepsDeleted(ctx)
		if err != nil {
			t.Fatalf("Operators still running: %v", err)
		}
	})
}

func runsOnCiOperator() bool {
	_, ok := os.LookupEnv("OPENSHIFT_BUILD_NAMESPACE")
	return ok
}

