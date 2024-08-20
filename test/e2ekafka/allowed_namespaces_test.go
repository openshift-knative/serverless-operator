package e2ekafka

import (
	kafkav1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/e2e"
	"testing"
)

func TestKnativeKafkaCannotBeInstalledInARandomNamespace(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	knativeKafkaGvr := kafkav1alpha1.SchemeGroupVersion.WithResource("knativekafkas")
	e2e.VerifyCRCannotBeInstalledInRandomNamespace(t, caCtx, test.Namespace, knativeKafkaGvr, "KnativeKafka", "knative-kafka")
}
