package e2e

import (
	"testing"

	eventinge2e "github.com/openshift-knative/knative-eventing-operator/test/e2e"
)

func TestEventingOperator(t *testing.T) {
	for _, spec := range eventinge2e.Specifications() {
		t.Run(spec.Name, spec.Func)
	}
}
