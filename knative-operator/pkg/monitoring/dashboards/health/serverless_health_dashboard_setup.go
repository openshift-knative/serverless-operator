package health

import (
	"context"
	"errors"
	"fmt"
	"os"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/dashboards"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var logh = common.Log.WithName("health dashboard")

func InstallHealthDashboard(api client.Client) error {
	namespace := os.Getenv(common.NamespaceEnvKey)
	if namespace == "" {
		return errors.New("NAMESPACE not provided via environment")
	}
	err := api.Get(context.TODO(), client.ObjectKey{Name: dashboards.ConfigManagedNamespace}, &corev1.Namespace{})
	if apierrors.IsNotFound(err) {
		logh.Info(fmt.Sprintf("namespace %q not found. Skipping to create dashboard.", dashboards.ConfigManagedNamespace))
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get namespace %q: %w", dashboards.ConfigManagedNamespace, err)
	}
	deploymentName, err := monitoring.GetOperatorDeploymentName()
	if err != nil {
		return err
	}
	manifest, err := manifest(api, deploymentName, namespace)
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

// manifest returns dashboard resources manifest
func manifest(apiclient client.Client, deploymentName string, namespace string) (mf.Manifest, error) {
	manifest, err := mfc.NewManifest(manifestPath(), apiclient, mf.UseLogger(logh.WithName("mf")))
	if err != nil {
		return mf.Manifest{}, fmt.Errorf("failed to read dashboard manifest: %w", err)
	}
	transforms := []mf.Transformer{
		common.SetAnnotations(map[string]string{
			common.ServerlessOperatorOwnerName:      deploymentName,
			common.ServerlessOperatorOwnerNamespace: namespace,
		}),
		mf.InjectNamespace(dashboards.ConfigManagedNamespace),
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
	path := os.Getenv("DASHBOARDS_ROOT_MANIFEST_PATH")
	return path + "/grafana-dash-knative-health.yaml"
}
