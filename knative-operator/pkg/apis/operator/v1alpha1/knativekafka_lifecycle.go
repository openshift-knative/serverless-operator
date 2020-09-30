package v1alpha1

import (
	knativeoperatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/apis"
)

var (
	kafkaCondSet = apis.NewLivingConditionSet(
		knativeoperatorv1alpha1.InstallSucceeded,
	)
)

// IsReady looks at the conditions returns true if they are all true.
func (is *KnativeKafkaStatus) IsReady() bool {
	return kafkaCondSet.Manage(is).IsHappy()
}

// MarkInstallSucceeded marks the InstallationSucceeded status as true.
func (is *KnativeKafkaStatus) MarkInstallSucceeded() {
	kafkaCondSet.Manage(is).MarkTrue(knativeoperatorv1alpha1.InstallSucceeded)
}

// MarkInstallFailed marks the InstallationSucceeded status as false with the given
// message.
func (is *KnativeKafkaStatus) MarkInstallFailed(msg string) {
	kafkaCondSet.Manage(is).MarkFalse(
		knativeoperatorv1alpha1.InstallSucceeded,
		"Error",
		"Install failed with message: %s", msg)
}
