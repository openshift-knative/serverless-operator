package consoleutil

import (
	"sync/atomic"

	configv1 "github.com/openshift/api/config/v1"
)

const ConsoleClusterOperatorName = "console"

var consoleInstalled = atomic.Bool{}

// SetConsoleToInstalledStatus updates to true the detected status of the console capability.
// Once a capability is installed it cannot be uninstalled.
func SetConsoleToInstalledStatus() {
	consoleInstalled.Store(true)
}

// IsConsoleInstalled checks the detected status of the console capability.
func IsConsoleInstalled() bool {
	return consoleInstalled.Load()
}

// IsClusterOperatorAvailable iterates over conditions of the related resource
// and checks if it is available.
func IsClusterOperatorAvailable(status configv1.ClusterOperatorStatus) bool {
	for _, cond := range status.Conditions {
		if cond.Type == configv1.OperatorAvailable && (cond.Status == configv1.ConditionTrue || cond.Reason == "Unmanaged") {
			return true
		}
	}
	return false
}
