package servicemesh

import (
	"context"

	mf "github.com/jcrossley3/manifestival"
	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceMeshControlPlane name
const smcpName = "basic-install"

var log = common.Log.WithName("servicemesh")

// Delete deletes SMCP deployed by previous version.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	if err := deleteSMCP(instance, api); err != nil {
		return err
	}
	if err := deleteNetworkPolicies(instance, api); err != nil {
		return err
	}
	return deleteNetworkingIstio(instance, api)
}

func deleteSMCP(instance *servingv1alpha1.KnativeServing, api client.Client) error {
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

func deleteNetworkPolicies(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Deleting Network Policies")
	const path = "deploy/resources/networkpolicies.yaml"

	manifest, err := mf.NewManifest(path, false, api)
	if err != nil {
		return err
	}
	transforms := []mf.Transformer{mf.InjectNamespace(instance.GetNamespace())}
	if err := manifest.Transform(transforms...); err != nil {
		return err
	}
	return manifest.DeleteAll()
}

func deleteNetworkingIstio(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Deleting networking-istio deployment")

	deploy := &appsv1.Deployment{}
	if err := api.Get(context.TODO(), client.ObjectKey{Namespace: instance.GetNamespace(), Name: "networking-istio"}, deploy); err != nil {
		if apierrors.IsNotFound(err) {
			// We can safely ignore this. There is nothing to do for us.
			return nil
		} else if err != nil {
			return err
		}
	}
	return api.Delete(context.TODO(), deploy)
}
