package knativeserving

import (
	"github.com/openshift-knative/serverless-operator/serving/operator/pkg/controller/knativeserving/openshift"
)

func init() {
	platforms = append(platforms, openshift.Configure)
}
