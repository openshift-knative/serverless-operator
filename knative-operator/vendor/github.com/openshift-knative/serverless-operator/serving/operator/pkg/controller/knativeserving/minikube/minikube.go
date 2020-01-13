package minikube

import (
	"context"
	"fmt"

	mf "github.com/jcrossley3/manifestival"
	servingv1alpha1 "github.com/openshift-knative/serverless-operator/serving/operator/pkg/apis/serving/v1alpha1"
	"github.com/openshift-knative/serverless-operator/serving/operator/pkg/controller/knativeserving/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	extension = common.Extension{
		Transformers: []mf.Transformer{egress},
		PreInstalls:  []common.Extender{checkDependencies},
	}
	log = logf.Log.WithName("minikube")
	api client.Client
)

// Configure minikube if we're soaking in it
func Configure(c client.Client, _ *runtime.Scheme, _ *mf.Manifest) (*common.Extension, error) {
	node := &v1.Node{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: "minikube"}, node); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Unable to query for minikube node")
		}
		// Not running in minikube
		return nil, nil
	}
	api = c
	return &extension, nil
}

func egress(u *unstructured.Unstructured) error {
	if u.GetKind() == "ConfigMap" && u.GetName() == "config-network" {
		data := map[string]string{
			"istio.sidecar.includeOutboundIPRanges": "10.0.0.1/24",
		}
		common.UpdateConfigMap(u, data, log)
	}
	return nil
}

// Check for all dependencies
func checkDependencies(instance *servingv1alpha1.KnativeServing) error {
	istio := schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1alpha3", Kind: "gateway"}
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(istio)
	if err := api.List(context.TODO(), nil, list); err != nil {
		msg := fmt.Sprintf("Istio not detected, GVK %v missing", istio)
		instance.Status.MarkDependencyMissing(msg)
		log.Error(err, msg)
		return err
	}

	log.Info("All dependencies are installed")
	instance.Status.MarkDependenciesInstalled()
	return nil
}
