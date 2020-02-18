package servicemesh

import (
	"context"

	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceMeshControlPlane name
const smcpName = "basic-install"

var log = common.Log.WithName("servicemesh")

// Delete deletes SMCP deployed by previous version.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Deleting SMCP")

	smcp := &maistrav1.ServiceMeshControlPlane{}
	if err := api.Get(context.TODO(), client.ObjectKey{Namespace: common.IngressNamespace(instance.GetNamespace()), Name: smcpName}, smcp); err != nil {
		if apierrors.IsNotFound(err) {
			// We can safely ignore this. There is nothing to do for us.
			return nil
		} else if err != nil {
			return err
		}
	}
	return api.Delete(context.TODO(), smcp)
}
