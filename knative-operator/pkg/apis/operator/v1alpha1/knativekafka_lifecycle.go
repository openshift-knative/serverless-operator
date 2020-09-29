package v1alpha1

import "knative.dev/pkg/apis"

const (
	// DependenciesInstalled is a Condition indicating that potential dependencies have
	// been installed correctly.
	DependenciesInstalled apis.ConditionType = "DependenciesInstalled"
	// InstallSucceeded is a Condition indiciating that the installation of the component
	// itself has been successful.
	InstallSucceeded apis.ConditionType = "InstallSucceeded"
	// DeploymentsAvailable is a Condition indicating whether or not the Deployments of
	// the respective component have come up successfully.
	DeploymentsAvailable apis.ConditionType = "DeploymentsAvailable"
)

var (
	kafkaCondSet = apis.NewLivingConditionSet(
		DependenciesInstalled,
		DeploymentsAvailable,
		InstallSucceeded,
	)
)

// IsReady looks at the conditions returns true if they are all true.
func (is *KnativeKafkaStatus) IsReady() bool {
	return kafkaCondSet.Manage(is).IsHappy()
}

// MarkDependencyInstalling marks the DependenciesInstalled status as false with the
// given message.
func (is *KnativeKafkaStatus) MarkDependencyInstalling(msg string) {
	kafkaCondSet.Manage(is).MarkFalse(
		DependenciesInstalled,
		"Installing",
		"Dependency installing: %s", msg)
}

// MarkDependenciesInstalled marks the DependenciesInstalled status as true.
func (is *KnativeKafkaStatus) MarkDependenciesInstalled() {
	kafkaCondSet.Manage(is).MarkTrue(DependenciesInstalled)
}
