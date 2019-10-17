package e2e

import (
	"github.com/openshift-knative/knative-eventing-operator/test"
)

// Specifications - specififactions list
func Specifications() []test.Specification {
	return specifications
}

var specifications []test.Specification = []test.Specification{
	test.NewSpec("TestKnativeEventingInstall", testKnativeEventingInstall),
}
