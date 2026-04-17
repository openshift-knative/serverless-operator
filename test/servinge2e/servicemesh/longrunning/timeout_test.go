package longrunning

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress/resources"
	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e/servicemesh"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/spoof"
	servingTest "knative.dev/serving/test"
)

const (
	routeTimeout = "800"
	sleepTime    = 630000
)

func TestTimeoutForLongRunningRequests(t *testing.T) {
	ctx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
	defer test.CleanupAll(t, ctx)

	service := test.Service("longrunning", test.Namespace, pkgTest.ImagePath(test.AutoscaleImg), map[string]string{
		servicemesh.ServingEnablePassthroughKey: "true",
		resources.SetRouteTimeoutAnnotation:     routeTimeout,
	}, nil)
	service = test.WithServiceReadyOrFail(ctx, service)
	serviceURL := service.Status.URL.URL()
	serviceURL.RawQuery = fmt.Sprintf("sleep=%d", sleepTime)

	if _, err := pkgTest.WaitForEndpointStateWithTimeout(
		context.Background(),
		ctx.Clients.Kube,
		t.Logf,
		serviceURL,
		spoof.MatchesBody("Slept"),
		"CheckResponse",
		true,
		time.Second*900,
		servingTest.AddRootCAtoTransport(context.Background(), t.Logf, &servingTest.Clients{KubeClient: ctx.Clients.Kube}, true),
	); err != nil {
		t.Fatalf("Unexpected state for %s :%v", serviceURL, err)
	}
}
