package test

import (
	"context"
	"fmt"
	"time"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
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
	OLMNamespace              = "openshift-marketplace"
	ServerlessOperatorPackage = "serverless-operator"
)

func UpdateSubscriptionChannelSource(ctx *Context, name, channel, source string) (*operatorsv1alpha1.Subscription, error) {
	patch := []byte(fmt.Sprintf(`{"spec":{"channel":"%s","source":"%s"}}`, channel, source))
	return ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).
		Patch(context.Background(), name, types.MergePatchType, patch, metav1.PatchOptions{})
}

func WaitForClusterServiceVersionState(ctx *Context, name, namespace string, inState func(s *operatorsv1alpha1.ClusterServiceVersion, err error) (bool, error)) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	var lastState *operatorsv1alpha1.ClusterServiceVersion
	var err error
	waitErr := wait.PollUntilContextTimeout(context.Background(), Interval, Timeout, true, func(_ context.Context) (bool, error) {
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

func DeleteClusterServiceVersion(ctx *Context, name, namespace string) error {
	return ctx.Clients.OLM.OperatorsV1alpha1().ClusterServiceVersions(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}

func DeleteSubscription(ctx *Context, name, namespace string) error {
	return ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}

func Subscription(subscriptionName, startingCSV string) *operatorsv1alpha1.Subscription {
	return &operatorsv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       operatorsv1alpha1.SubscriptionKind,
			APIVersion: operatorsv1alpha1.SubscriptionCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: OperatorsNamespace,
			Name:      subscriptionName,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			CatalogSource:          Flags.CatalogSource,
			CatalogSourceNamespace: OLMNamespace,
			Package:                ServerlessOperatorPackage,
			Channel:                Flags.Channel,
			InstallPlanApproval:    operatorsv1alpha1.ApprovalManual,
			StartingCSV:            startingCSV,
		},
	}
}

func CreateSubscription(ctx *Context, name, startingCSV string) (*operatorsv1alpha1.Subscription, error) {
	subs, err := ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).Create(context.Background(), Subscription(name, startingCSV), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func CreateOperatorGroup(ctx *Context, name, namespace string) (*operatorsv1.OperatorGroup, error) {
	return ctx.Clients.OLM.OperatorsV1().OperatorGroups(namespace).Create(context.Background(),
		&operatorsv1.OperatorGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}, metav1.CreateOptions{})
}
