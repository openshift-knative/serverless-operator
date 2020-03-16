package framework

import (
	"fmt"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	operatorsv1 "github.com/openshift/api/operator/v1"
	consoleapi "github.com/openshift/console-operator/pkg/api"
)

// set the operator config to a pristine state to start a next round of tests
// this should by default nullify out any customizations a user sets
func Pristine(t *testing.T, client *ClientSet) (*operatorsv1.Console, error) {
	t.Helper()
	operatorConfig, err := client.Operator.Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	copy := operatorConfig.DeepCopy()
	cleanSpec := operatorsv1.ConsoleSpec{}
	// we can set a default management state & log level, but
	// nothing else should be necessary
	cleanSpec.ManagementState = operatorsv1.Managed
	cleanSpec.LogLevel = operatorsv1.Normal
	copy.Spec = cleanSpec
	return client.Operator.Consoles().Update(copy)
}

func MustPristine(t *testing.T, client *ClientSet) *operatorsv1.Console {
	t.Helper()
	operatorConfig, err := Pristine(t, client)
	if err != nil {
		t.Fatal(err)
	}
	return operatorConfig
}

func isOperatorManaged(cr *operatorsv1.Console) bool {
	return cr.Spec.ManagementState == operatorsv1.Managed
}

func isOperatorUnmanaged(cr *operatorsv1.Console) bool {
	return cr.Spec.ManagementState == operatorsv1.Unmanaged
}

func isOperatorRemoved(cr *operatorsv1.Console) bool {
	return cr.Spec.ManagementState == operatorsv1.Removed
}

type operatorStateReactionFn func(cr *operatorsv1.Console) bool

func ensureConsoleIsInDesiredState(t *testing.T, client *ClientSet, state operatorsv1.ManagementState) error {
	t.Helper()
	var operatorConfig *operatorsv1.Console
	// var checkFunc func()
	var checkFunc operatorStateReactionFn

	switch state {
	case operatorsv1.Managed:
		checkFunc = isOperatorManaged
	case operatorsv1.Unmanaged:
		checkFunc = isOperatorUnmanaged
	case operatorsv1.Removed:
		checkFunc = isOperatorRemoved
	}

	err := wait.Poll(1*time.Second, AsyncOperationTimeout, func() (stop bool, err error) {
		operatorConfig, err = client.Operator.Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return checkFunc(operatorConfig), nil
	})
	if err != nil {
		DumpObject(t, "the latest observed state of the console resource", operatorConfig)
		DumpOperatorLogs(t, client)
		return fmt.Errorf("failed to wait to change console operator state to 'Removed': %s", err)
	}
	return nil
}

func manageConsole(t *testing.T, client *ClientSet) error {
	t.Helper()
	operatorConfig, err := client.Operator.Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if isOperatorManaged(operatorConfig) {
		t.Logf("console operator already in 'Managed' state")
		return nil
	}

	t.Logf("changing console operator state to 'Managed'...")

	_, err = client.Operator.Consoles().Patch(consoleapi.ConfigResourceName, types.MergePatchType, []byte(`{"spec": {"managementState": "Managed"}}`))
	if err != nil {
		return err
	}
	if err := ensureConsoleIsInDesiredState(t, client, operatorsv1.Managed); err != nil {
		return fmt.Errorf("unable to change console operator state to 'Managed': %s", err)
	}

	err = ConsoleResourcesAvailable(client)
	if err != nil {
		t.Fatal(err)
	}

	return nil
}

func unmanageConsole(t *testing.T, client *ClientSet) error {
	t.Helper()
	operatorConfig, err := client.Operator.Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if isOperatorUnmanaged(operatorConfig) {
		t.Logf("console operator already in 'Unmanaged' state")
		return nil
	}

	t.Logf("changing console operator state to 'Unmanaged'...")

	_, err = client.Operator.Consoles().Patch(consoleapi.ConfigResourceName, types.MergePatchType, []byte(`{"spec": {"managementState": "Unmanaged"}}`))
	if err != nil {
		return err
	}
	if err := ensureConsoleIsInDesiredState(t, client, operatorsv1.Unmanaged); err != nil {
		return fmt.Errorf("unable to change console operator state to 'Unmanaged': %s", err)
	}

	return nil
}

func removeConsole(t *testing.T, client *ClientSet) error {
	t.Helper()
	operatorConfig, err := client.Operator.Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if isOperatorRemoved(operatorConfig) {
		t.Logf("console operator already in 'Removed' state")
		return nil
	}

	t.Logf("changing console operator state to 'Removed'...")

	_, err = client.Operator.Consoles().Patch(consoleapi.ConfigResourceName, types.MergePatchType, []byte(`{"spec": {"managementState": "Removed"}}`))
	if err != nil {
		return err
	}
	if err := ensureConsoleIsInDesiredState(t, client, operatorsv1.Removed); err != nil {
		return fmt.Errorf("unable to change console operator state to 'Removed': %s", err)
	}

	return nil
}

func MustManageConsole(t *testing.T, client *ClientSet) error {
	t.Helper()
	if err := manageConsole(t, client); err != nil {
		t.Fatal(err)
	}
	return nil
}

func MustUnmanageConsole(t *testing.T, client *ClientSet) error {
	t.Helper()
	if err := unmanageConsole(t, client); err != nil {
		t.Fatal(err)
	}
	return nil
}

func MustRemoveConsole(t *testing.T, client *ClientSet) error {
	t.Helper()
	if err := removeConsole(t, client); err != nil {
		t.Fatal(err)
	}
	return nil
}

func MustNormalLogLevel(t *testing.T, client *ClientSet) error {
	t.Helper()
	operatorConfig, err := client.Operator.Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("checking if console operator LogLevel is set to 'Normal'...")
	if operatorConfig.Spec.LogLevel == operatorsv1.Normal {
		return nil
	}
	err = SetLogLevel(t, client, operatorsv1.Normal)
	if err != nil {
		t.Fatal(err)
	}
	return nil
}

func SetLogLevel(t *testing.T, client *ClientSet, logLevel operatorsv1.LogLevel) error {
	t.Helper()
	operatorConfig, err := client.Operator.Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	deployment, err := GetConsoleDeployment(client)
	if err != nil {
		return err
	}
	currentDeploymentGeneration := deployment.ObjectMeta.Generation
	currentOperatorConfigGeneration := operatorConfig.ObjectMeta.Generation

	t.Logf("setting console operator to '%s' LogLevel ...", logLevel)
	_, err = client.Operator.Consoles().Patch(consoleapi.ConfigResourceName, types.MergePatchType, []byte(fmt.Sprintf(`{"spec": {"logLevel": "%s"}}`, logLevel)))
	if err != nil {
		return err
	}

	err = wait.PollImmediate(1*time.Second, 1*time.Minute, func() (bool, error) {
		newOperatorConfig, err := client.Operator.Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
		newDeployment, err := GetConsoleDeployment(client)
		if err != nil {
			return false, nil
		}
		if GenerationChanged(newOperatorConfig.ObjectMeta.Generation, currentOperatorConfigGeneration) {
			return false, nil
		}
		if GenerationChanged(newDeployment.ObjectMeta.Generation, currentDeploymentGeneration) {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func GenerationChanged(oldGeneration, newGeneration int64) bool {
	return oldGeneration == newGeneration
}

type conditionsMap = map[string]operatorsv1.OperatorCondition

func operatorConditionsMap(operatorConfig *operatorsv1.Console) conditionsMap {
	conditions := make(conditionsMap, len(operatorConfig.Status.Conditions))
	for _, condition := range operatorConfig.Status.Conditions {
		conditions[condition.Type] = condition
	}
	return conditions
}

var (
	deprecatedConditions = []string{"Failing"}
)

// we may want our tests to also tell us if deprecated conditions are still being set
func reportDeprecatedConditions(t *testing.T, conditions conditionsMap) {
	for _, dep := range deprecatedConditions {
		if _, ok := conditions[dep]; ok {
			t.Logf("Deprecated condition %s still exists \n", dep)
		}
	}
}

// the operator is settled if all custom conditions with suffixes match:
// - *Available: true
// - *Progressing: false
// - *Degraded: false
func operatorIsSettled(operatorConfig *operatorsv1.Console) (settled bool, unmetConditions []string) {
	settled = true
	conditions := operatorConditionsMap(operatorConfig)
	unmetConditions = []string{}

	tests := []struct {
		name           string
		conditionType  string
		expectedStatus operatorsv1.ConditionStatus
	}{
		{
			name:           "Degraded suffix conditions must be false",
			conditionType:  operatorsv1.OperatorStatusTypeDegraded,
			expectedStatus: operatorsv1.ConditionFalse,
		}, {
			name:           "Progressing suffix conditions should be False",
			conditionType:  operatorsv1.OperatorStatusTypeProgressing,
			expectedStatus: operatorsv1.ConditionFalse,
		}, {
			name:           "Available suffix conditions should be True",
			conditionType:  operatorsv1.OperatorStatusTypeAvailable,
			expectedStatus: operatorsv1.ConditionTrue,
		},
	}
	for _, test := range tests {
		for _, condition := range conditions {
			if strings.HasSuffix(condition.Type, test.conditionType) {
				// any condition with a matching suffix must match status else we are not settled
				if condition.Status != test.expectedStatus {
					unmetConditions = append(unmetConditions, condition.Type)
					settled = false
				}
			}
		}
	}
	// returning unmet conditions for logging purposes
	return settled, unmetConditions
}

func operatorIsObservingCurrentGeneration(t *testing.T, operatorConfig *operatorsv1.Console) bool {
	if operatorConfig.Status.ObservedGeneration != operatorConfig.ObjectMeta.Generation {
		t.Logf("waiting for observed generation %d to match generation %d... \n", operatorConfig.Status.ObservedGeneration, operatorConfig.ObjectMeta.Generation)
		return false
	}
	return true
}

// A helper to ensure our operator config reaches a settled state before we
// begin the next test.
func WaitForSettledState(t *testing.T, client *ClientSet, phase string) (settled bool, err error) {
	t.Helper()
	t.Logf("waiting for %s to reach settled state...\n", phase)
	// don't rush it
	interval := 2 * time.Second
	// it should never take this long for a test to pass
	max := 240 * time.Second
	count := 0
	unmetConditions := []string{}
	pollErr := wait.Poll(interval, max, func() (stop bool, err error) {
		// lets be informed about tests that take a long time to settle
		count++
		logUnsettledAtInterval(t, count)
		operatorConfig, err := client.Operator.Consoles().Get(consoleapi.ConfigResourceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		// first, wait until we are observing the correct generation. if we are still looking at a previous
		// generation, pass on this iteration of the loop
		isCurrentGen := operatorIsObservingCurrentGeneration(t, operatorConfig)
		if !isCurrentGen {
			return false, nil
		}
		// then wait until the operator status settle
		settled, unmet := operatorIsSettled(operatorConfig)
		unmetConditions = unmet // avoid shadow
		return settled, nil
	})
	if pollErr != nil {
		t.Errorf("operator has not reached settled state in %v attempts due to %v - %v", max, unmetConditions, pollErr)
	}
	return true, nil

}

// short term helper to simply print how long it takes to reach a settled state at certain intervals
func logUnsettledAtInterval(t *testing.T, count int) {
	// arbitrary steps at which to print a notification
	steps := []int{10, 30, 60, 90, 120, 180, 200}
	for _, step := range steps {
		if count == step {
			t.Logf("waited %d seconds to reach settled state...\n", count)
		}
	}
}
