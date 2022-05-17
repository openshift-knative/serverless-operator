package test

import (
	"context"
	"fmt"
	"time"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

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

func DeleteSubscription(ctx *Context, name, namespace string) error {
	return ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}

func Subscription(subscriptionName string) *operatorsv1alpha1.Subscription {
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
			//TODO: Pass this as a flag, similar to --csv=serverless-operator.v1.23.0
			StartingCSV: "serverless-operator.v1.22.0",
		},
	}
}

func CreateSubscription(ctx *Context, name string) (*operatorsv1alpha1.Subscription, error) {
	subs, err := ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).Create(context.Background(), Subscription(name), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	//ctx.AddToCleanup(func() error {
	//	ctx.T.Logf("Cleaning up Subscription '%s/%s'", subs.Namespace, subs.Name)
	//	return ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(subs.Namespace).Delete(context.Background(), subs.Name, metav1.DeleteOptions{})
	//})
	return subs, nil
}

//func WaitForSubscriptionState(ctx *Context, name, namespace string, inState func(s *operatorsv1alpha1.Subscription, err error) (bool, error)) (*operatorsv1alpha1.Subscription, error) {
//	var lastState *operatorsv1alpha1.Subscription
//	var err error
//	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
//		lastState, err = ctx.Clients.OLM.OperatorsV1alpha1().Subscriptions(namespace).Get(context.Background(), name, metav1.GetOptions{})
//		return inState(lastState, err)
//	})
//
//	if waitErr != nil {
//		return lastState, fmt.Errorf("subscription %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
//	}
//	return lastState, nil
//}
//
//func IsSubscriptionInstalledCSVPresent(s *operatorsv1alpha1.Subscription, err error) (bool, error) {
//	return s.Status.InstalledCSV != "" && s.Status.InstalledCSV != "<none>", err
//}
