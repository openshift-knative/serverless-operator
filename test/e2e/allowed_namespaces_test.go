package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"knative.dev/operator/pkg/apis/operator"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestServingCannotBeInstalledInARandomNamespace(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	VerifyCRCannotBeInstalledInRandomNamespace(t, caCtx, test.Namespace, operator.KnativeServingResource.WithVersion(v1beta1.SchemaVersion), operator.KindKnativeServing, "knative-serving", "knative-serving")
}

func TestEventingCannotBeInstalledInARandomNamespace(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	VerifyCRCannotBeInstalledInRandomNamespace(t, caCtx, test.Namespace, operator.KnativeEventingResource.WithVersion(v1beta1.SchemaVersion), operator.KindKnativeEventing, "knative-eventing", "knative-eventing")
}
