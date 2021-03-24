package test

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	apierrs "k8s.io/apimachinery/pkg/api/errors"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
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

func Subscription(subscriptionName string) *v1alpha1.Subscription {
	//namespace, name, catalogSourceName, packageName, channel string, approval v1alpha1.Approval
	return &v1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.SubscriptionKind,
			APIVersion: v1alpha1.SubscriptionCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: OperatorsNamespace,
			Name:      subscriptionName,
		},
		Spec: &v1alpha1.SubscriptionSpec{
			CatalogSource:          Flags.CatalogSource,
			CatalogSourceNamespace: OLMNamespace,
			Package:                ServerlessOperatorPackage,
			Channel:                Flags.Channel,
			InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
		},
	}
}

func WithOperatorReady(ctx *Context, subscriptionName string) (*v1alpha1.Subscription, error) {
	if _, err := CreateSubscription(ctx, subscriptionName); err != nil {
		return nil, err
	}

	subs, err := WaitForSubscriptionState(ctx, subscriptionName, OperatorsNamespace, IsSubscriptionInstalledCSVPresent)
	if err != nil {
		return nil, err
	}

	csvName := subs.Status.InstalledCSV

	csv, err := WaitForClusterServiceVersionState(ctx, csvName, OperatorsNamespace, IsCSVSucceeded)
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up CSV '%s/%s'", csv.Namespace, csv.Name)
		return ctx.Clients.OLM.OperatorsV1alpha1().ClusterServiceVersions(csv.Namespace).Delete(context.Background(), csv.Name, metav1.DeleteOptions{})
	})

	return subs, nil
}

func CreateSubscription(ctx *Context, name string) (*v1alpha1.Subscription, error) {
	subs, err := ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).Create(context.Background(), Subscription(name), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Subscription '%s/%s'", subs.Namespace, subs.Name)
		return ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(subs.Namespace).Delete(context.Background(), subs.Name, metav1.DeleteOptions{})
	})
	return subs, nil
}

func WaitForSubscriptionState(ctx *Context, name, namespace string, inState func(s *v1alpha1.Subscription, err error) (bool, error)) (*v1alpha1.Subscription, error) {
	var lastState *v1alpha1.Subscription
	var err error
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("subscription %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func UpdateSubscriptionChannelSource(ctx *Context, name, channel, source string) (*v1alpha1.Subscription, error) {
	patch := []byte(fmt.Sprintf(`{"spec":{"channel":"%s","source":"%s"}}`, channel, source))
	return ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).
		Patch(context.Background(), name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
}

func WaitForClusterServiceVersionState(ctx *Context, name, namespace string, inState func(s *v1alpha1.ClusterServiceVersion, err error) (bool, error)) (*v1alpha1.ClusterServiceVersion, error) {
	var lastState *v1alpha1.ClusterServiceVersion
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

func IsCSVSucceeded(c *v1alpha1.ClusterServiceVersion, err error) (bool, error) {
	// The CSV might not exist yet.
	if apierrs.IsNotFound(err) {
		return false, nil
	}
	return c.Status.Phase == "Succeeded", err
}

func IsSubscriptionInstalledCSVPresent(s *v1alpha1.Subscription, err error) (bool, error) {
	return s.Status.InstalledCSV != "" && s.Status.InstalledCSV != "<none>", err
}
