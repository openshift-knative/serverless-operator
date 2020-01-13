package knativeserving

import (
	"github.com/openshift-knative/serverless-operator/serving/operator/pkg/controller/knativeserving/minikube"
)

func init() {
	platforms = append(platforms, minikube.Configure)
}
