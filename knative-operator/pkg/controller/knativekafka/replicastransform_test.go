package knativekafka

import (
	"testing"
)

func TestCheckHAComponent(t *testing.T) {
	cases := []struct {
		name           string
		deploymentName string
		shouldFail     bool
	}{{
		name:           "kafka channel controller",
		deploymentName: "kafka-ch-controller",
		shouldFail:     false,
	}, {
		name:           "kafka webhook",
		deploymentName: "kafka-webhook",
		shouldFail:     false,
	}, {
		name:           "kafka source controller",
		deploymentName: "kafka-controller",
		shouldFail:     false,
	}, {
		name:           "kafka channel dispatcher",
		deploymentName: "kafka-ch-dispatcher",
		shouldFail:     true,
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checkHAComponent(tc.deploymentName)
			if result == tc.shouldFail {
				t.Errorf("Got: %v, want: %v\n", result, tc.shouldFail)
			}
		})
	}
}
