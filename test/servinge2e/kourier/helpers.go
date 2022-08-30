package kourier

import (
	"context"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1alpha1 "knative.dev/serving/pkg/apis/serving/v1alpha1"
)

const (
	image          = "quay.io/openshift-knative-serving-test/helloworld:v1.3"
	helloworldText = "Hello World!"
)

func withDomainMappingReadyOrFail(ctx *test.Context, dm *servingv1alpha1.DomainMapping) *servingv1alpha1.DomainMapping {
	dm, err := ctx.Clients.Serving.ServingV1alpha1().DomainMappings(dm.Namespace).Create(context.Background(), dm, metav1.CreateOptions{})
	if err != nil {
		ctx.T.Fatalf("Error creating ksvc: %v", err)
	}

	// Let the ksvc be deleted after test
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Knative Service '%s/%s'", dm.Namespace, dm.Name)
		return ctx.Clients.Serving.ServingV1alpha1().DomainMappings(dm.Namespace).Delete(context.Background(), dm.Name, metav1.DeleteOptions{})
	})

	dm, err = test.WaitForDomainMappingState(ctx, dm.Name, dm.Namespace, test.IsDomainMappingReady)
	if err != nil {
		ctx.T.Fatalf("Error waiting for ksvc readiness: %v", err)
	}

	return dm
}
