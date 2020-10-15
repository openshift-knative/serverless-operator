package common

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyEnvironmentToDeployment adds/removes the specified values in the map to the environment
// variables of the specified deployment.
// NotFound errors are ignored.
func ApplyEnvironmentToDeployment(namespace, name string, env map[string]string, c client.Client) error {
	deploy := &appsv1.Deployment{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: namespace}, deploy); err != nil {
		// We ignore NotFound errors.
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to fetch deployment: %w", err)
	}

	before := deploy.DeepCopy()
	for c := range deploy.Spec.Template.Spec.Containers {
		for k, v := range env {
			// If value is not empty then update deployment controller with env
			if v != "" {
				deploy.Spec.Template.Spec.Containers[c].Env = AppendUnique(deploy.Spec.Template.Spec.Containers[c].Env, k, v)
			} else {
				// If value is empty then remove those keys from deployment controller
				deploy.Spec.Template.Spec.Containers[c].Env = remove(deploy.Spec.Template.Spec.Containers[c].Env, k)
			}
		}
	}

	// Only update if we actually changed something.
	if !equality.Semantic.DeepEqual(before.Spec.Template.Spec, deploy.Spec.Template.Spec) {
		log.Info("Updating environment of deployment", "namespace", namespace, "name", name, "env", env)
		if err := c.Update(context.TODO(), deploy); err != nil {
			return fmt.Errorf("failed to update deployment with new environment: %w", err)
		}
	}

	return nil
}

func remove(env []v1.EnvVar, key string) []v1.EnvVar {
	for i := range env {
		if env[i].Name == key {
			return append(env[:i], env[i+1:]...)
		}
	}
	return env
}

func AppendUnique(orgEnv []v1.EnvVar, key, value string) []v1.EnvVar {
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
