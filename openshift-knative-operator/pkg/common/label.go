package common

import (
	mf "github.com/manifestival/manifestival"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// InjectEnvironmentIntoDeployment injects the common label into the resources.
func InjectCommonLabel() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		labels := u.GetLabels()
		if labels == nil {
			labels = make(map[string]string, 1)
		}
		labels[socommon.ServerlessCommonLabelKey] = socommon.ServerlessCommonLabelValue
		u.SetLabels(labels)
		return nil
	}
}
