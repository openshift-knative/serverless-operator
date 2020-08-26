package dashboard

import (
	"context"
	"fmt"
	"os"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = common.Log.WithName("dashboard")

const ConfigManagedNamespace = "openshift-config-managed"

// Apply applies dashboard resources.
func Apply(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	err := api.Get(context.TODO(), client.ObjectKey{Name: ConfigManagedNamespace}, &corev1.Namespace{})
	if apierrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("namespace %q not found. Skipping to create dashboard.", ConfigManagedNamespace))
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get namespace %q: %w", ConfigManagedNamespace, err)
	}

	manifest, err := manifest(instance, api)
	if err != nil {
		return fmt.Errorf("failed to load dashboard manifest: %w", err)
	}
	log.Info("Installing dashboard")
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply dashboard manifest: %w", err)
	}
	log.Info("Dashboard is ready")
	return nil
}

// Delete deletes dashboard resources.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Deleting dashboard")
	manifest, err := manifest(instance, api)
	if err != nil {
		return fmt.Errorf("failed to load dashboard manifest: %w", err)
	}

	if err := manifest.Delete(); err != nil {
		return fmt.Errorf("failed to delete dashboard manifest: %w", err)
	}
	return nil
}

// manifest returns dashboard deploymnet resources manifest
func manifest(instance *servingv1alpha1.KnativeServing, apiclient client.Client) (mf.Manifest, error) {
	manifest, err := mfc.NewManifest(manifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return mf.Manifest{}, fmt.Errorf("failed to read dashboard manifest: %w", err)
	}

	// set owner to watch events.
	transforms := []mf.Transformer{mf.InjectNamespace(ConfigManagedNamespace), common.SetOwnerAnnotations(instance)}

	manifest, err = manifest.Transform(transforms...)
	if err != nil {
		return mf.Manifest{}, fmt.Errorf("failed to transform kn dashboard resources manifest: %w", err)
	}
	return manifest, nil
}

// manifestPath returns dashboard resource manifest path
func manifestPath() string {
	path := os.Getenv("DASHBOARD_MANIFEST_PATH")
	if path == "" {
		return "deploy/resources/dashboards/grafana-dash-knative.yaml"
	}
	return path
}
