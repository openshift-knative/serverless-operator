package common

import (
	"context"
	"fmt"
	"os"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var logh = Log.WithName("health dashboard")

const ConfigManagedNamespace = "openshift-config-managed"

func InstallHealthDashboard(api client.Client) error {
	namespace, err := getOperatorNamespace()
	if err != nil {
		return err
	}
	instance, err := getServerlessOperatorDeployment(api, namespace)
	if err != nil {
		return err
	}
	err = api.Get(context.TODO(), client.ObjectKey{Name: ConfigManagedNamespace}, &corev1.Namespace{})
	if apierrors.IsNotFound(err) {
		logh.Info(fmt.Sprintf("namespace %q not found. Skipping to create dashboard.", ConfigManagedNamespace))
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get namepsace %q: %w", ConfigManagedNamespace, err)
	}

	manifest, err := manifest(instance, api, namespace)
	if err != nil {
		return fmt.Errorf("failed to load dashboard manifest: %w", err)
	}
	logh.Info("Installing dashboard")
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply dashboard manifest: %w", err)
	}
	logh.Info("Dashboard is ready")
	return nil
}

// manifest returns dashboard deploymnet resources manifest
func manifest(instance *appsv1.Deployment, apiclient client.Client, namespace string) (mf.Manifest, error) {
	manifest, err := mfc.NewManifest(manifestPath(), apiclient, mf.UseLogger(logh.WithName("mf")))
	if err != nil {
		return mf.Manifest{}, fmt.Errorf("failed to read dashboard manifest: %w", err)
	}

	instance.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})

	instance.SetNamespace(namespace)

	transforms := []mf.Transformer{mf.InjectOwner(instance)}
	if ConfigManagedNamespace != "" {
		transforms = append(transforms, mf.InjectNamespace(ConfigManagedNamespace))
	}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		return mf.Manifest{}, fmt.Errorf("unable to transform role and roleBinding serviceMonitor manifest %w", err)
	}
	manifest, err = manifest.Transform(transforms...)
	if err != nil {
		return mf.Manifest{}, fmt.Errorf("failed to transform kn dashboard resources manifest %w", err)
	}
	return manifest, nil
}

// manifestPath returns health dashboard resource manifest path
func manifestPath() string {
	path := os.Getenv("HEALTH_DASHBOARD_MANIFEST_PATH")
	if path == "" {
		return "deploy/resources/dashboards/grafana-dash-knative-health.yaml"
	}
	return path
}
