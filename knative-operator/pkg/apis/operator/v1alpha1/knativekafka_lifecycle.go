package v1alpha1

import "knative.dev/pkg/apis"
import knativeoperatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"

var (
	kafkaCondSet = apis.NewLivingConditionSet(
		knativeoperatorv1alpha1.DependenciesInstalled,
		knativeoperatorv1alpha1.DeploymentsAvailable,
		knativeoperatorv1alpha1.InstallSucceeded,
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
		knativeoperatorv1alpha1.DependenciesInstalled,
		"Installing",
		"Dependency installing: %s", msg)
}

// MarkDependenciesInstalled marks the DependenciesInstalled status as true.
func (is *KnativeKafkaStatus) MarkDependenciesInstalled() {
	kafkaCondSet.Manage(is).MarkTrue(knativeoperatorv1alpha1.DependenciesInstalled)
}
