package status

import (
	"bytes"
	"fmt"
	"strings"

	v1 "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"

	"k8s.io/klog"

	operatorsv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/console-operator/pkg/console/errors"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
)

// Operator status is set on the Operator Config, which is:
//   group: console.operator.openshift.io
//   kind:  Console
//   name:  console
// And is replicated out onto the the clusteroperator by a separate sync loop:
//   group: config.openshift.io
//   kind:  ClusterOperator
//   name:  console
//

// handleDegraded(), handleProgressing(), handleAvailable() each take a typePrefix string representing "category"
// and a reason string, representing the actual problem.
// the provided err will be used as the detailed message body if it is not nil
// note that available status is desired to be true, where degraded & progressing are desired to be false
// example:
//   c.handleDegraded(operatorConfig, "RouteStatus", "FailedHost", error.New("route is not available at canonical host..."))
// generates:
//   - Type RouteStatusDegraded
//     Status: true
//     Reason: Failedhost
//     Message: error string value is used as message
// all degraded suffix conditions will be aggregated into a final "Degraded" status that will be set on the console ClusterOperator
func HandleDegraded(operatorConfig *operatorsv1.Console, typePrefix string, reason string, err error) {
	conditionType := typePrefix + operatorsv1.OperatorStatusTypeDegraded
	handleCondition(operatorConfig, conditionType, reason, err)
}

func HandleProgressing(operatorConfig *operatorsv1.Console, typePrefix string, reason string, err error) {
	conditionType := typePrefix + operatorsv1.OperatorStatusTypeProgressing
	handleCondition(operatorConfig, conditionType, reason, err)
}

func HandleAvailable(operatorConfig *operatorsv1.Console, typePrefix string, reason string, err error) {
	conditionType := typePrefix + operatorsv1.OperatorStatusTypeAvailable
	handleCondition(operatorConfig, conditionType, reason, err)
}

// HandleProgressingOrDegraded exists until we remove type SyncError
// If isSyncError
// - Type suffix will be set to Progressing
// if it is any other kind of error
// - Type suffix will be set to Degraded
// TODO: when we eliminate the special case SyncError, this helper can go away.
// When we do that, however, we must make sure to register deprecated conditions with NewRemoveStaleConditions()
func HandleProgressingOrDegraded(operatorConfig *operatorsv1.Console, typePrefix string, reason string, err error) {
	if errors.IsSyncError(err) {
		HandleDegraded(operatorConfig, typePrefix, reason, nil)
		HandleProgressing(operatorConfig, typePrefix, reason, err)
	} else {
		HandleDegraded(operatorConfig, typePrefix, reason, err)
		HandleProgressing(operatorConfig, typePrefix, reason, nil)
	}
}

func handleCondition(operatorConfig *operatorsv1.Console, conditionTypeWithSuffix string, reason string, err error) {
	if err != nil {
		klog.Errorln(conditionTypeWithSuffix, reason, err.Error())
		v1helpers.SetOperatorCondition(&operatorConfig.Status.Conditions,
			operatorsv1.OperatorCondition{
				Type:    conditionTypeWithSuffix,
				Status:  setConditionValue(conditionTypeWithSuffix, err),
				Reason:  reason,
				Message: err.Error(),
			})
		return
	}
	v1helpers.SetOperatorCondition(&operatorConfig.Status.Conditions,
		operatorsv1.OperatorCondition{
			Type:   conditionTypeWithSuffix,
			Status: setConditionValue(conditionTypeWithSuffix, err),
		})
}

// Available is an inversion of the other conditions
func setConditionValue(conditionType string, err error) operatorsv1.ConditionStatus {
	if strings.HasSuffix(conditionType, operatorsv1.OperatorStatusTypeAvailable) {
		if err != nil {
			return operatorsv1.ConditionFalse
		}
		return operatorsv1.ConditionTrue
	}
	if err != nil {
		return operatorsv1.ConditionTrue
	}
	return operatorsv1.ConditionFalse
}

func IsDegraded(operatorConfig *operatorsv1.Authentication) bool {
	for _, condition := range operatorConfig.Status.Conditions {
		if strings.HasSuffix(condition.Type, operatorsv1.OperatorStatusTypeDegraded) &&
			condition.Status == operatorsv1.ConditionTrue {
			return true
		}
	}
	return false
}

// Lets transition to using this, and get the repetition out of all of the above.
func SyncStatus(operatorConfigClient v1.ConsoleInterface, operatorConfig *operatorsv1.Console) (*operatorsv1.Console, error) {
	logConditions(operatorConfig.Status.Conditions)
	updatedConfig, err := operatorConfigClient.UpdateStatus(operatorConfig)
	if err != nil {
		errMsg := fmt.Errorf("status update error: %v", err)
		klog.Error(errMsg)
		return nil, errMsg
	}
	return updatedConfig, nil
}

// Outputs the condition as a log message based on the detail of the condition in the form of:
//   Status.Condition.<Condition>: <Bool>
//   Status.Condition.<Condition>: <Bool> (<Reason>)
//   Status.Condition.<Condition>: <Bool> (<Reason>) <Message>
//   Status.Condition.<Condition>: <Bool> <Message>
func logConditions(conditions []operatorsv1.OperatorCondition) {
	klog.V(4).Infoln("Operator.Status.Conditions")

	for _, condition := range conditions {
		buf := bytes.Buffer{}
		buf.WriteString(fmt.Sprintf("Status.Condition.%s: %s", condition.Type, condition.Status))
		hasMessage := condition.Message != ""
		hasReason := condition.Reason != ""
		if hasMessage && hasReason {
			buf.WriteString(" |")
			if hasReason {
				buf.WriteString(fmt.Sprintf(" (%s)", condition.Reason))
			}
			if hasMessage {
				buf.WriteString(fmt.Sprintf(" %s", condition.Message))
			}
		}
		klog.V(4).Infoln(buf.String())
	}
}
