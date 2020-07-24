package common

import (
	"context"
	"fmt"
	"os"

	"github.com/coreos/go-semver/semver"
	configv1 "github.com/openshift/api/config/v1"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Validator func(context.Context, client.Client, v1alpha1.KComponent) (bool, string, error)

// Validate runs through a variety of checks for the instance
func Validate(ctx context.Context, c client.Client, instance v1alpha1.KComponent) (allowed bool, reason string, err error) {
	log := Log.WithName("validation")
	stages := []Validator{
		validateNamespace,
		validateVersion,
		validateLoneliness,
	}
	for _, stage := range stages {
		allowed, reason, err = stage(ctx, c, instance)
		if len(reason) > 0 {
			if err != nil {
				log.Error(err, reason)
			} else {
				log.Info(reason)
			}
		}
		if !allowed {
			return
		}
	}
	return
}

// validate minimum openshift version
func validateVersion(ctx context.Context, c client.Client, _ v1alpha1.KComponent) (bool, string, error) {
	version, present := os.LookupEnv("MIN_OPENSHIFT_VERSION")
	if !present {
		return true, "", nil
	}
	minVersion, err := semver.NewVersion(version)
	if err != nil {
		return false, "Unable to validate version; check MIN_OPENSHIFT_VERSION env var", nil
	}

	clusterVersion := &configv1.ClusterVersion{}
	if err := c.Get(ctx, client.ObjectKey{Name: "version"}, clusterVersion); err != nil {
		return false, "Unable to get ClusterVersion", err
	}

	current, err := semver.NewVersion(clusterVersion.Status.Desired.Version)
	if err != nil {
		return false, "Could not parse version string", err
	}

	if current.Major == 0 && current.Minor == 0 {
		return true, "CI build detected, bypassing version check", nil
	}

	if current.LessThan(*minVersion) {
		msg := fmt.Sprintf("Version constraint not fulfilled: minimum version: %s, current version: %s", minVersion.String(), current.String())
		return false, msg, nil
	}
	return true, "", nil
}

// validate required namespace, if any
func validateNamespace(ctx context.Context, _ client.Client, instance v1alpha1.KComponent) (bool, string, error) {
	env := "REQUIRED_SERVING_NAMESPACE"
	if _, ok := instance.(*v1alpha1.KnativeEventing); ok {
		env = "REQUIRED_EVENTING_NAMESPACE"
	}
	ns, required := os.LookupEnv(env)
	if required && ns != instance.GetNamespace() {
		return false, fmt.Sprintf("Instance may only be created in %s namespace", ns), nil
	}
	return true, "", nil
}

// validate this is the only instance in the cluster
// TODO: this, but less type-safely
func validateLoneliness(ctx context.Context, c client.Client, instance v1alpha1.KComponent) (bool, string, error) {
	switch v := instance.(type) {
	case *v1alpha1.KnativeServing:
		return validateLonelyServing(ctx, c, v)
	case *v1alpha1.KnativeEventing:
		return validateLonelyEventing(ctx, c, v)
	}
	return false, "Unsupported type", nil
}

// validate this is the only KS in the cluster
// TODO: not this
func validateLonelyServing(ctx context.Context, c client.Client, ks *v1alpha1.KnativeServing) (bool, string, error) {
	list := &v1alpha1.KnativeServingList{}
	if err := c.List(ctx, &client.ListOptions{Namespace: ""}, list); err != nil {
		return false, "Unable to list instances", err
	}
	for _, item := range list.Items {
		if ks.Name != item.Name || ks.Namespace != item.Namespace {
			return false, fmt.Sprintf("Existing instance found: %s/%s", item.Namespace, item.Name), nil
		}
	}
	return true, "", nil
}

// validate this is the only KE in the cluster
// TODO: not this
func validateLonelyEventing(ctx context.Context, c client.Client, ke *v1alpha1.KnativeEventing) (bool, string, error) {
	list := &v1alpha1.KnativeEventingList{}
	if err := c.List(ctx, &client.ListOptions{Namespace: ""}, list); err != nil {
		return false, "Unable to list KnativeEventings", err
	}
	for _, item := range list.Items {
		if ke.Name != item.Name || ke.Namespace != item.Namespace {
			return false, fmt.Sprintf("Existing instance found: %s/%s", item.Namespace, item.Name), nil
		}
	}
	return true, "", nil
}
