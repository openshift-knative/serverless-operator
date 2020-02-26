package common

import (
	"context"
	"fmt"

	mf "github.com/jcrossley3/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	MutationTimestampKey = "knative-serving-openshift/mutation"
	kourierPath          = "deploy/resources/kourier/kourier-latest.yaml"
)

var Log = logf.Log.WithName("knative").WithName("openshift")

// Configure is a  helper to set a value for a key, potentially overriding existing contents.
func Configure(ks *servingv1alpha1.KnativeServing, cm, key, value string) bool {
	if ks.Spec.Config == nil {
		ks.Spec.Config = map[string]map[string]string{}
	}

	old, found := ks.Spec.Config[cm][key]
	if found && value == old {
		return false
	}

	if ks.Spec.Config[cm] == nil {
		ks.Spec.Config[cm] = map[string]string{}
	}

	ks.Spec.Config[cm][key] = value
	Log.Info("Configured", "map", cm, key, value, "old value", old)
	return true
}

// IngressNamespace returns namespace where ingress is deployed.
func IngressNamespace(servingNamespace string) string {
	return servingNamespace + "-ingress"
}

// KourierManifest gets kourier manifest after injecting knative serving ingress namespace.
func KourierManifest(instance *servingv1alpha1.KnativeServing, api client.Client) (mf.Manifest, error) {
	manifest, err := mf.NewManifest(kourierPath, false, api)
	if err != nil {
		return manifest, err
	}
	transforms := []mf.Transformer{mf.InjectNamespace(IngressNamespace(instance.GetNamespace()))}
	if err := manifest.Transform(transforms...); err != nil {
		return manifest, err
	}
	return manifest, nil
}

// CheckDeployments checks for deployments in ingress namespace.
// This function is copied from knativeserving_controller.go in serving-operator.
func CheckDeployments(manifest *mf.Manifest, instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Checking deployments")
	for _, u := range manifest.Resources {
		if u.GetKind() == "Deployment" {
			deployment := &appsv1.Deployment{}
			err := api.Get(context.TODO(), client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}, deployment)
			if err != nil {
				return err
			}
			for _, c := range deployment.Status.Conditions {
				if c.Type == appsv1.DeploymentAvailable && c.Status != v1.ConditionTrue {
					return fmt.Errorf("Deployment %q/%q not ready", u.GetName(), u.GetNamespace())
				}
			}
		}
	}
	return nil
}
