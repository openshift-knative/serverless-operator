// +build e2e

package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	upstreame2e "knative.dev/serving-operator/test/e2e"
	upstreamtest "knative.dev/serving-operator/test"
)

func TestUpstreamKnativeServingOperator(t *testing.T) {
	upstreamtest.ServingOperatorNamespace = "knative-serving"
	suite := upstreame2e.ComplianceSuite()
	ctx := test.SetupClusterAdmin(t)

	defer test.CleanupAll(ctx)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(ctx) })

	createSubscriptionAndWaitForCSVtoSucceed(t, ctx)

	skipConfigureSubTest := upstreamtest.Skip(
		"TestKnativeServingDeployment/configure",
		"Skip due to SRVKS-241")

	upstreamtest.
		NewContext(t).
		WithOverride(skipConfigureSubTest).
		RunSuite(suite)

	undeployServerlessOperatorAndCheckDependentOperatorsRemoved(t, ctx)
}
