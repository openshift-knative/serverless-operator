package common

import (
	mf "github.com/manifestival/manifestival"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// InjectCommonLabelIntoNamespace injects the common label into the namespaces.
func InjectCommonLabelIntoNamespace() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Namespace" {
			return nil
		}
		labels := u.GetLabels()
		if labels == nil {
			labels = make(map[string]string, 1)
		}
		labels[socommon.ServerlessCommonLabelKey] = socommon.ServerlessCommonLabelValue
		u.SetLabels(labels)
		return nil
	}
}
