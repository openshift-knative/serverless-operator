package common

import (
	"context"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyProxySettings updates Knative controller env to use cluster wide proxy information
func ApplyProxySettings(ks *servingv1alpha1.KnativeServing, c client.Client) error {
	var proxyEnv = map[string]string{
		"HTTP_PROXY": os.Getenv("HTTP_PROXY"),
		"NO_PROXY":   os.Getenv("NO_PROXY"),
	}
	deploy := &appsv1.Deployment{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: "controller", Namespace: ks.GetNamespace()}, deploy); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	for c := range deploy.Spec.Template.Spec.Containers {
		for k, v := range proxyEnv {
			// If value is not empty then update deployment controller with env
			if v != "" {
				deploy.Spec.Template.Spec.Containers[c].Env = appendUnique(deploy.Spec.Template.Spec.Containers[c].Env, k, v)
			} else {
				// If value is empty then remove those keys from deployment controller
				deploy.Spec.Template.Spec.Containers[c].Env = remove(deploy.Spec.Template.Spec.Containers[c].Env, k)
			}
		}
	}
	return c.Update(context.TODO(), deploy)
}

func remove(env []v1.EnvVar, key string) []v1.EnvVar {
	for i := range env {
		if env[i].Name == key {
			return append(env[:i], env[i+1:]...)
		}
	}
	return env
}

func appendUnique(orgEnv []v1.EnvVar, key, value string) []v1.EnvVar {
	// Set the value if the key is already present.
	for i := range orgEnv {
		if orgEnv[i].Name == key {
			orgEnv[i].Value = value
			return orgEnv
		}
	}
	// If not, append a key-value pair.
	return append(orgEnv, v1.EnvVar{
		Name:  key,
		Value: value,
	})
}
