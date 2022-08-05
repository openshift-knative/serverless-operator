package knativeeventing

import (
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/dashboards"
)

var (
	ke = &operatorv1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-eventing",
			Namespace: "knative-eventing",
		},
	}
	req = reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: ke.Namespace, Name: ke.Name},
	}
	dashboardNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: dashboards.ConfigManagedNamespace,
		},
	}
	eventingNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "knative-eventing",
		},
	}
)

func init() {
	os.Setenv("OPERATOR_NAME", "TEST_OPERATOR")
	os.Setenv(dashboards.DashboardsManifestPathEnvVar, "../../../deploy/resources/dashboards")

	apis.AddToScheme(scheme.Scheme)
}
