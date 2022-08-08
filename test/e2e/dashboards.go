package e2e

import (
	"context"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift-knative/serverless-operator/test"
)

var (
	EventingDashboards = []string{
		"grafana-dashboard-definition-knative-eventing-resources",
		"grafana-dashboard-definition-knative-eventing-broker",
		"grafana-dashboard-definition-knative-eventing-kafka-broker",
		"grafana-dashboard-definition-knative-eventing-source",
		"grafana-dashboard-definition-knative-eventing-channel",
		"grafana-dashboard-definition-knative-eventing-kafka-sink",
	}
)

func VerifyDashboards(t *testing.T, caCtx *test.Context, dashboards []string) {
	t.Run("Verify dashboards", func(t *testing.T) {
		t.Parallel()

		ns := "openshift-config-managed"
		ctx := context.Background()

		for _, d := range dashboards {
			t.Run(d, func(t *testing.T) {
				err := wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
					_, err := caCtx.Clients.Kube.CoreV1().ConfigMaps(ns).Get(ctx, d, metav1.GetOptions{})
					if err != nil && !apierrors.IsNotFound(err) {
						return false, err
					}
					if apierrors.IsNotFound(err) {
						return false, nil
					}
					return true, nil
				})
				if err != nil {
					t.Error(err)
				}
			})
		}
	})
}
