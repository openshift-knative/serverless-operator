package test

import (
	"time"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// Interval specifies the time between two polls.
	Interval = 10 * time.Second
	// Timeout specifies the timeout for the function PollImmediate to reach a certain status.
	Timeout                   = 5 * time.Minute
	OperatorsNamespace        = "openshift-operators"
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
			CatalogSource:          ServerlessOperatorPackage,
			CatalogSourceNamespace: OLMNamespace,
			Package:                ServerlessOperatorPackage,
			Channel:                "techpreview",
			InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
		},
	}
}

func WithOperatorReady(ctx *Context, subscriptionName string) (*v1alpha1.Subscription, error) {
	_, err := CreateSubscription(ctx, subscriptionName)

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
		return ctx.Clients.OLM.OperatorsV1alpha1().ClusterServiceVersions(OperatorsNamespace).Delete(csv.Name, &metav1.DeleteOptions{})
	})

	return subs, nil
}

func CreateSubscription(ctx *Context, name string) (*v1alpha1.Subscription, error) {
	subs, err := ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).Create(Subscription(name))
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		return ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).Delete(subs.Name, &metav1.DeleteOptions{})
	})
	return subs, nil
}

func WaitForSubscriptionState(ctx *Context, name, namespace string, inState func(s *v1alpha1.Subscription, err error) (bool, error)) (*v1alpha1.Subscription, error) {
	var lastState *v1alpha1.Subscription
	var err error
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(namespace).Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "subscription %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func WaitForClusterServiceVersionState(ctx *Context, name, namespace string, inState func(s *v1alpha1.ClusterServiceVersion, err error) (bool, error)) (*v1alpha1.ClusterServiceVersion, error) {
	var lastState *v1alpha1.ClusterServiceVersion
	var err error
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.OLM.OperatorsV1alpha1().ClusterServiceVersions(namespace).Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "clusterserviceversion %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func IsCSVSucceeded(c *v1alpha1.ClusterServiceVersion, err error) (bool, error) {
	return c.Status.Phase == "Succeeded", err
}

func IsSubscriptionInstalledCSVPresent(s *v1alpha1.Subscription, err error) (bool, error) {
	return s.Status.InstalledCSV != "" && s.Status.InstalledCSV != "<none>", err
}
