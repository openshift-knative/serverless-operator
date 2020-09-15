package dashboard

import (
	"context"
	"fmt"
	"os"
	"strings"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = common.Log.WithName("dashboard")

const ConfigManagedNamespace = "openshift-config-managed"

const ServingDashboardPath = "deploy/resources/dashboards/grafana-dash-knative.yaml"
const EventingBrokerDashboardPath = "deploy/resources/dashboards/grafana-dash-knative-eventing-broker.yaml"
const EventingSourceDashboardPath = "deploy/resources/dashboards/grafana-dash-knative-eventing-source.yaml"

// Apply applies dashboard resources.
func Apply(path string, owner mf.Transformer, api client.Client) error {
	err := api.Get(context.TODO(), client.ObjectKey{Name: ConfigManagedNamespace}, &corev1.Namespace{})
	if apierrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("namespace %q not found. Skipping to create dashboard.", ConfigManagedNamespace))
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get namespace %q: %w", ConfigManagedNamespace, err)
	}
	manifest, err := manifest(path, owner, api)
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
func Delete(path string, owner mf.Transformer, api client.Client) error {
	log.Info("Deleting dashboard")
	manifest, err := manifest(path, owner, api)
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
	manifest, err := mfc.NewManifest(manifestPath(path), apiclient, mf.UseLogger(log.WithName("mf")))
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

// manifestPath returns dashboard resource manifest path
func manifestPath(defaultPath string) string {

	// meant for testing only, if not in testing mode use the
	// default path.
	pathServing := os.Getenv("TEST_DASHBOARD_MANIFEST_PATH")
	pathSource := os.Getenv("TEST_SOURCE_DASHBOARD_MANIFEST_PATH")
	pathBroker := os.Getenv("TEST_BROKER_DASHBOARD_MANIFEST_PATH")

	if pathSource != "" {
		if strings.Contains(defaultPath, "eventing-source.yaml") {
			return pathSource
		}
	}
	if pathBroker != "" {
		if strings.Contains(defaultPath, "eventing-broker.yaml") {
			return pathBroker
		}
	}
	if pathServing != "" {
		return pathServing
	}
	return defaultPath
}
