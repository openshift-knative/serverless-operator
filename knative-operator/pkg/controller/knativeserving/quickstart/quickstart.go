package quickstart

import (
	"fmt"
	"os"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	apierrs "k8s.io/apimachinery/pkg/api/meta"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnvKey is the environment variable that decides which manifest to load
const EnvKey = "QUICKSTART_MANIFEST_PATH"

var log = common.Log.WithName("quickstart")

// Apply applies Quickstart resources.
func Apply(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	manifest, err := mfc.NewManifest(manifestPath(), api, mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return fmt.Errorf("failed to load quickstart manifest: %w", err)
	}

	log.Info("Installing Quickstarts")
	if err := manifest.Apply(); err != nil {
		if apierrs.IsNoMatchError(err) {
			log.Info("ConsoleQuickStart CRD not installed, skipping quickstart installation")
			return nil
		}
		return fmt.Errorf("failed to apply quickstart manifest: %w", err)
	}
	log.Info("Quickstarts installed")
	return nil
}

// Delete deletes Quickstart resources.
func Delete(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Deleting Quickstarts")
	manifest, err := mfc.NewManifest(manifestPath(), api, mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return fmt.Errorf("failed to load quickstart manifest: %w", err)
	}

	if err := manifest.Delete(); err != nil {
		if apierrs.IsNoMatchError(err) {
			log.Info("ConsoleQuickStart CRD not installed, skipping quickstart installation")
			return nil
		}
		return fmt.Errorf("failed to delete quickstart manifest: %w", err)
	}
	return nil
}

func manifestPath() string {
	return os.Getenv(EnvKey)
}
