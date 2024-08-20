package e2e

import (
	"github.com/openshift-knative/serverless-operator/test"
	"knative.dev/operator/pkg/apis/operator"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"testing"
)

func TestServingCannotBeInstalledInARandomNamespace(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	VerifyCRCannotBeInstalledInRandomNamespace(t, caCtx, test.Namespace, operator.KnativeServingResource.WithVersion(v1beta1.SchemaVersion), operator.KindKnativeServing, "knative-serving")
}

func TestEventingCannotBeInstalledInARandomNamespace(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	VerifyCRCannotBeInstalledInRandomNamespace(t, caCtx, test.Namespace, operator.KnativeEventingResource.WithVersion(v1beta1.SchemaVersion), operator.KindKnativeEventing, "knative-eventing")
}
