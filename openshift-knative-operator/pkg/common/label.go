package common

import (
	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// InjectEnvironmentIntoDeployment injects the common label into the resources.
func InjectCommonLabel() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		labels := u.GetLabels()
		if labels == nil {
			labels = make(map[string]string, 1)
		}
		labels["app.openshift.io/part-of"] = "openshift-serverless"
		u.SetLabels(labels)
		return nil
	}
}
