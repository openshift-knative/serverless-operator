package dashboards

import (
	"context"
	"fmt"
	"os"

	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = common.Log.WithName("dashboard")

const ConfigManagedNamespace = "openshift-config-managed"
const DashboardsManifestPathEnvVar = "DASHBOARDS_ROOT_MANIFEST_PATH"

// Apply applies dashboard resources under the manifestSubPath directory.
func Apply(manifestSubPath string, instance base.KComponent, api client.Client) error {
	err := api.Get(context.TODO(), client.ObjectKey{Name: ConfigManagedNamespace}, &corev1.Namespace{})
	if apierrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("namespace %q not found. Skipping to create dashboard.", ConfigManagedNamespace))
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get namespace %q: %w", ConfigManagedNamespace, err)
	}
	manifest, err := manifest(getDashboardsPath(manifestSubPath), getAnnotationsFromInstance(instance), api)
	if err != nil {
		return fmt.Errorf("failed to load dashboards manifests: %w", err)
	}
	log.Info("Installing dashboards under ", "path:", getDashboardsPath(manifestSubPath))
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply dashboards manifests: %w", err)
	}
	log.Info("Dashboards are ready")
	return nil
}

// Delete deletes dashboard resources.
func Delete(manifestSubPath string, instance base.KComponent, api client.Client) error {
	manifest, err := manifest(getDashboardsPath(manifestSubPath), getAnnotationsFromInstance(instance), api)
	if err != nil {
		return fmt.Errorf("failed to load dashboards manifests: %w", err)
	}
	log.Info("Deleting dashboards under ", "path:", getDashboardsPath(manifestSubPath))
	if err := manifest.Delete(); err != nil {
		return fmt.Errorf("failed to delete dashboards manifests: %w", err)
	}
	return nil
}

// manifest returns dashboards resources manifest
func manifest(path string, owner mf.Transformer, apiclient client.Client) (mf.Manifest, error) {
	manifest, err := mfc.NewManifest(path, apiclient, mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return mf.Manifest{}, fmt.Errorf("failed to read dashboards manifests: %w", err)
	}

	// set owner to watch events.
	transforms := []mf.Transformer{mf.InjectNamespace(ConfigManagedNamespace), owner}

	manifest, err = manifest.Transform(transforms...)
	if err != nil {
		return mf.Manifest{}, fmt.Errorf("failed to transform kn dashboard resource manifests: %w", err)
	}
	return manifest, nil
}

func getAnnotationsFromInstance(instance base.KComponent) mf.Transformer {
	switch instance.(type) {
	case *operatorv1alpha1.KnativeEventing:
		return common.SetAnnotations(map[string]string{
			socommon.EventingOwnerName:      instance.GetName(),
			socommon.EventingOwnerNamespace: instance.GetNamespace(),
		})
	case *operatorv1alpha1.KnativeServing:
		return common.SetAnnotations(map[string]string{
			socommon.ServingOwnerName:      instance.GetName(),
			socommon.ServingOwnerNamespace: instance.GetNamespace(),
		})
	}
	return nil
}

func getDashboardsPath(subPath string) string {
	path := os.Getenv(DashboardsManifestPathEnvVar)
	return path + "/" + subPath
}
