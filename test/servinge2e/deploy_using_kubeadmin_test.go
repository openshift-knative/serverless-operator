package servinge2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
)

const (
	testNamespace         = "serverless-tests"
	testNamespace2        = "serverless-tests2"
	image                 = "gcr.io/knative-samples/helloworld-go"
	helloworldService     = "helloworld-go"
	helloworldService2    = "helloworld-go2"
	kubeHelloworldService = "kube-helloworld-go"
	helloworldText        = "Hello World!"
)

func TestDeploymentUsingKubeadmin(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	if _, err := test.WithServiceReady(caCtx, helloworldService, testNamespace, image); err != nil {
		t.Fatal("Knative Service not ready", err)
	}
}
