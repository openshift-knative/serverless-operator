package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
)

const (
	knativeServing = "knative-serving"
	testNamespace  = "serverless-tests"
)

func TestDeployment(t *testing.T) {
	ctx := test.Setup(t)
	defer ctx.Cleanup()
	test.CleanupOnInterrupt(t, func() { ctx.Cleanup() })

	t.Log("Deploying Serverless Operator")
	_, err := test.WithOperatorReady(ctx, "serverless-operator-subscription")
	if err != nil {
		t.Fatal("Failed", err)
	}

	t.Log("Deploying KnativeServing")
	_, err = test.WithKnativeServingReady(ctx, knativeServing, knativeServing)
	if err != nil {
		t.Fatal("Failed to deploy KnativeServing", err)
	}

	t.Log("Deploying Knative Service")
	image := "gcr.io/knative-samples/helloworld-go"
	_, err = test.WithServiceReady(ctx, "helloworld-go", testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	ctx.Cleanup()

	err = test.WaitForOperatorDepsDeleted(ctx)
	if err != nil {
		t.Fatalf("Operators still running: %v", err)
	}
}
