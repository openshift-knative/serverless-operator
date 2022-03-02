package common

import (
	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var commonLabel = map[string]string{"app.openshift.io/part-of": "openshift-serverless"}

// InjectEnvironmentIntoDeployment injects the common label into the resources.
func InjectCommonLabel() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		u.SetLabels(commonLabel)
		return nil
	}
}
