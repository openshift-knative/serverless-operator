package v1alpha1

import (
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/pkg/apis"
)

var (
	kafkaCondSet = apis.NewLivingConditionSet(
		base.DeploymentsAvailable,
		base.InstallSucceeded,
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
	kafkaCondSet.Manage(is).MarkTrue(base.InstallSucceeded)
}

// MarkInstallFailed marks the InstallationSucceeded status as false with the given
// message.
func (is *KnativeKafkaStatus) MarkInstallFailed(msg string) {
	kafkaCondSet.Manage(is).MarkFalse(
		base.InstallSucceeded,
		"Error",
		"Install failed with message: %s", msg)
}

// MarkDeploymentsAvailable marks the DeploymentsAvailable status as true.
func (is *KnativeKafkaStatus) MarkDeploymentsAvailable() {
	kafkaCondSet.Manage(is).MarkTrue(base.DeploymentsAvailable)
}

// MarkDeploymentsNotReady marks the DeploymentsAvailable status as false and calls out
// it's waiting for deployments.
func (is *KnativeKafkaStatus) MarkDeploymentsNotReady() {
	kafkaCondSet.Manage(is).MarkFalse(
		base.DeploymentsAvailable,
		"NotReady",
		"Waiting on deployments")
}
