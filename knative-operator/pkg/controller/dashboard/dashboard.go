package dashboard

import (
	"context"
	"fmt"

	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = common.Log.WithName("dashboard")

const ConfigManagedNamespace = "openshift-config-managed"

const EventingResourceDashboardPathEnvVar = "EVENTING_RESOURCES_DASHBOARD_MANIFEST_PATH"
const EventingBrokerDashboardPathEnvVar = "EVENTING_BROKER_DASHBOARD_MANIFEST_PATH"
const EventingSourceDashboardPathEnvVar = "EVENTING_SOURCE_DASHBOARD_MANIFEST_PATH"
const EventingChannelDashboardPathEnvVar = "EVENTING_CHANNEL_DASHBOARD_MANIFEST_PATH"
const ServingResourceDashboardPathEnvVar = "SERVING_RESOURCES_DASHBOARD_MANIFEST_PATH"

// Apply applies dashboard resources.
func Apply(path string, instance operatorv1alpha1.KComponent, api client.Client) error {
	err := api.Get(context.TODO(), client.ObjectKey{Name: ConfigManagedNamespace}, &corev1.Namespace{})
	if apierrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("namespace %q not found. Skipping to create dashboard.", ConfigManagedNamespace))
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get namespace %q: %w", ConfigManagedNamespace, err)
	}
	manifest, err := manifest(path, getAnnotationsFromInstance(instance), api)
	if err != nil {
		return fmt.Errorf("failed to load dashboard manifest: %w", err)
	}
	log.Info("Installing dashboard ", "path:", path)
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply dashboard manifest: %w", err)
	}
	log.Info("Dashboard is ready")
	return nil
}

// Delete deletes dashboard resources.
func Delete(path string, instance operatorv1alpha1.KComponent, api client.Client) error {
	log.Info("Deleting dashboard")
	manifest, err := manifest(path, getAnnotationsFromInstance(instance), api)
	if err != nil {
		return fmt.Errorf("failed to load dashboard manifest: %w", err)
	}

	if err := manifest.Delete(); err != nil {
		return fmt.Errorf("failed to delete dashboard manifest: %w", err)
	}
	return nil
}

// manifest returns dashboard deploymnet resources manifest
func manifest(path string, owner mf.Transformer, apiclient client.Client) (mf.Manifest, error) {
	manifest, err := mfc.NewManifest(path, apiclient, mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return mf.Manifest{}, fmt.Errorf("failed to read dashboard manifest: %w", err)
	}

	// set owner to watch events.
	transforms := []mf.Transformer{mf.InjectNamespace(ConfigManagedNamespace), owner}

	manifest, err = manifest.Transform(transforms...)
	if err != nil {
		return mf.Manifest{}, fmt.Errorf("failed to transform kn dashboard resources manifest: %w", err)
	}
	return manifest, nil
}

func getAnnotationsFromInstance(instance operatorv1alpha1.KComponent) mf.Transformer {
	var value interface{} = instance
	switch v := value.(type) {
	case operatorv1alpha1.KnativeEventing:
		return common.SetAnnotations(map[string]string{
			common.EventingOwnerName:      v.Name,
			common.EventingOwnerNamespace: v.Namespace,
		})
	case operatorv1alpha1.KnativeServing:
		return common.SetAnnotations(map[string]string{
			common.ServingOwnerName:      v.Name,
			common.ServingOwnerNamespace: v.Namespace,
		})
	}
	return nil
}
