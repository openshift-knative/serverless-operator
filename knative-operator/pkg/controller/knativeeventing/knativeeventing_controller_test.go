package knativeeventing

import (
	"os"

	"k8s.io/client-go/kubernetes/scheme"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/dashboards"
)

func init() {
	os.Setenv("OPERATOR_NAME", "TEST_OPERATOR")
	os.Setenv(dashboards.DashboardsManifestPathEnvVar, "../../../deploy/resources/dashboards")

	apis.AddToScheme(scheme.Scheme)
}
