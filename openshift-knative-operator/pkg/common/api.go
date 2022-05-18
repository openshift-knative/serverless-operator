package common

import (
	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DowngradePodDisruptionBudget downgrade the API version to policy/v1beta1.
func DowngradePodDisruptionBudget() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "PodDisruptionBudget" {
			return nil
		}
		if u.GetAPIVersion() != "policy/v1" {
			return nil
		}
		u.SetAPIVersion("policy/v1beta1")
		return nil
	}
}
