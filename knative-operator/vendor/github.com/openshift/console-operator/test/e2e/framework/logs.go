package framework

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/davecgh/go-spew/spew"

	consoleapi "github.com/openshift/console-operator/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodLog []string

func (log PodLog) Contains(re *regexp.Regexp) bool {
	for _, line := range log {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

type PodSetLogs map[string]PodLog

func (psl PodSetLogs) Contains(re *regexp.Regexp) bool {
	for _, podlog := range psl {
		if podlog.Contains(re) {
			return true
		}
	}
	return false
}

func GetLogsByLabelSelector(client *ClientSet, namespace string, labelSelector *metav1.LabelSelector) (PodSetLogs, error) {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	podList, err := client.Core.Pods(namespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}

	podLogs := make(PodSetLogs)
	for _, pod := range podList.Items {
		var podLog PodLog
		log, err := client.Core.Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream()
		if err != nil {
			return nil, fmt.Errorf("failed to get logs for pod %s: %s", pod.Name, err)
		}
		r := bufio.NewReader(log)
		for {
			line, readErr := r.ReadSlice('\n')
			if len(line) > 0 || readErr == nil {
				podLog = append(podLog, string(line))
			}
			if readErr == io.EOF {
				break
			} else if readErr != nil {
				return nil, fmt.Errorf("failed to read log for pod %s: %s", pod.Name, readErr)
			}
		}
		podLogs[pod.Name] = podLog
	}
	return podLogs, nil
}

// DumpObject prints the object to the test log.
func DumpObject(t *testing.T, prefix string, obj interface{}) {
	t.Logf("%s:\n%s", prefix, spew.Sdump(obj))
}

func DumpPodLogs(t *testing.T, podLogs PodSetLogs) {
	if len(podLogs) > 0 {
		for pod, logs := range podLogs {
			t.Logf("=== logs for pod/%s", pod)
			for _, line := range logs {
				t.Logf("%s", line)
			}
		}
		t.Logf("=== end of logs")
	}
}

func GetOperatorLogs(client *ClientSet) (PodSetLogs, error) {
	return GetLogsByLabelSelector(client, consoleapi.OpenShiftConsoleNamespace, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"name": "console-operator",
		},
	})
}

func DumpOperatorLogs(t *testing.T, client *ClientSet) {
	podLogs, err := GetOperatorLogs(client)
	if err != nil {
		t.Logf("failed to get the operator logs: %s", err)
	}
	DumpPodLogs(t, podLogs)
}
