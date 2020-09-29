package e2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	serviceName        = "knative-openshift-metrics"
	serviceMonitorName = serviceName
)

func TestServerlessOperator(t *testing.T) {
	caCtx := test.SetupClusterAdmin(t)

	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })

	t.Run("create subscription and wait for CSV to succeed", func(t *testing.T) {
		if _, err := test.WithOperatorReady(caCtx, "serverless-operator-subscription"); err != nil {
			t.Fatal("Failed", err)
		}
	})

	// Check the status of the serverless operator deployment
	if err := test.CheckDeploymentScale(caCtx, test.OperatorsNamespace, "knative-openshift", 1); err != nil {
		t.Fatalf("Failed to verify the operator deployment: %v", err)
	}

	// Check if service monitors are installed
	if _, err := caCtx.Clients.Kube.CoreV1().Services(test.OperatorsNamespace).Get(serviceName, metav1.GetOptions{}); err != nil {
		t.Fatalf("Failed to get the operator monitoring service : %v", err)
	}

	if _, err := caCtx.Clients.MonitoringClient.ServiceMonitors(test.OperatorsNamespace).Get(serviceMonitorName, metav1.GetOptions{}); err != nil {
		t.Fatalf("Failed to verify the operator service monitor: %v", err)
	}

	t.Run("undeploy serverless operator and check dependent monitoring resources removed", func(t *testing.T) {
		caCtx.Cleanup(t)
		if err := waitForOperatorMonitoringServiceDeleted(caCtx); err != nil {
			t.Fatalf("Monitoring service is still available: %v", err)
		}
		if err := waitForOperatorServiceMonitorDeleted(caCtx); err != nil {
			t.Fatalf("Service monitor is still available: %v", err)
		}
	})
}

func waitForOperatorMonitoringServiceDeleted(ctx *test.Context) error {
	waitErr := wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
		s, err := ctx.Clients.Kube.CoreV1().Services(test.OperatorsNamespace).Get(serviceName, metav1.GetOptions{})
		if err == nil && s != nil {
			return false, err
		}
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return true, err
	})
	if waitErr != nil {
		return errors.Wrapf(waitErr, "serverless operator monitoring dependencies not deleted in time")
	}
	return nil
}

func waitForOperatorServiceMonitorDeleted(ctx *test.Context) error {
	waitErr := wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
		sm, err := ctx.Clients.MonitoringClient.ServiceMonitors(test.OperatorsNamespace).Get(serviceMonitorName, metav1.GetOptions{})
		if err == nil && sm != nil {
			return false, err
		}
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return true, err
	})
	if waitErr != nil {
		return errors.Wrapf(waitErr, "serverless operator monitoring dependencies not deleted in time")
	}
	return nil
}
