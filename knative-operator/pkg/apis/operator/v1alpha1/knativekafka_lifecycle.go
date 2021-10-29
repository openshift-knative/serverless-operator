package v1alpha1

import (
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/apis"
)

var (
	kafkaCondSet = apis.NewLivingConditionSet(
		operatorv1alpha1.DeploymentsAvailable,
		operatorv1alpha1.InstallSucceeded,
	)
)

// InitializeConditions initializes conditions of an KnativeKafkaStatus
func (is *KnativeKafkaStatus) InitializeConditions() {
	kafkaCondSet.Manage(is).InitializeConditions()
}

// IsReady looks at the conditions returns true if they are all true.
func (is *KnativeKafkaStatus) IsReady() bool {
	return kafkaCondSet.Manage(is).IsHappy()
}

// MarkInstallSucceeded marks the InstallationSucceeded status as true.
func (is *KnativeKafkaStatus) MarkInstallSucceeded() {
	kafkaCondSet.Manage(is).MarkTrue(operatorv1alpha1.InstallSucceeded)
}

// MarkInstallFailed marks the InstallationSucceeded status as false with the given
// message.
func (is *KnativeKafkaStatus) MarkInstallFailed(msg string) {
	kafkaCondSet.Manage(is).MarkFalse(
		operatorv1alpha1.InstallSucceeded,
		"Error",
		"Install failed with message: %s", msg)
}

// MarkDeploymentsAvailable marks the DeploymentsAvailable status as true.
func (is *KnativeKafkaStatus) MarkDeploymentsAvailable() {
	kafkaCondSet.Manage(is).MarkTrue(operatorv1alpha1.DeploymentsAvailable)
}

// MarkDeploymentsNotReady marks the DeploymentsAvailable status as false and calls out
// it's waiting for deployments.
func (is *KnativeKafkaStatus) MarkDeploymentsNotReady() {
	kafkaCondSet.Manage(is).MarkFalse(
		operatorv1alpha1.DeploymentsAvailable,
		"NotReady",
		"Waiting on deployments")
}
