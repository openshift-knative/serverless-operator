package test

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	apierrs "k8s.io/apimachinery/pkg/api/errors"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// Interval specifies the time between two polls.
	Interval = 10 * time.Second
	// Timeout specifies the timeout for the function PollImmediate to reach a certain status.
	Timeout                   = 5 * time.Minute
	OperatorsNamespace        = "openshift-serverless"
)

func UpdateSubscriptionChannelSource(ctx *Context, name, channel, source string) (*operatorsv1alpha1.Subscription, error) {
	patch := []byte(fmt.Sprintf(`{"spec":{"channel":"%s","source":"%s"}}`, channel, source))
	return ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).
		Patch(context.Background(), name, types.MergePatchType, patch, metav1.PatchOptions{})
}

func WaitForClusterServiceVersionState(ctx *Context, name, namespace string, inState func(s *operatorsv1alpha1.ClusterServiceVersion, err error) (bool, error)) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	var lastState *operatorsv1alpha1.ClusterServiceVersion
	var err error
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.OLM.OperatorsV1alpha1().ClusterServiceVersions(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})
	if waitErr != nil {
		return lastState, fmt.Errorf("clusterserviceversion %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func IsCSVSucceeded(c *operatorsv1alpha1.ClusterServiceVersion, err error) (bool, error) {
	// The CSV might not exist yet.
	if apierrs.IsNotFound(err) {
		return false, nil
	}
	return c.Status.Phase == "Succeeded", err
}
