package functions

import (
	"fmt"
	"os"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = common.Log.WithName("function pipelines")

// InstallFunctionPipeline installs the Pipeline for on-cluster function builds
func InstallFunctionPipeline(api client.Client) error {
	manifest, err := mfc.NewManifest(pipelineManifestPath(), api, mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return fmt.Errorf("failed to read pipeline manifest: %w", err)
	}
	log.Info("Installing function pipeline")
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply pipeline manifest: %w", err)
	}
	return nil
}

// InstallFunctionTasks installs the Tasks for on-cluster function builds
func InstallFunctionTasks(api client.Client) error {
	manifest, err := mfc.NewManifest(tasksManifestPath(), api, mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return fmt.Errorf("failed to read task manifest: %w", err)
	}
	log.Info("Installing function build Tasks")
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply task manifest: %w", err)
	}
	return nil
}

// manifestPath returns function pipeline manifest path
func pipelineManifestPath() string {
	path := os.Getenv("FUNCTIONS_MANIFEST_PATH")
	return path + "/pipeline.yaml"
}

// manifestPath returns function build tasks manifest path
func tasksManifestPath() string {
	path := os.Getenv("FUNCTIONS_MANIFEST_PATH")
	return path + "/tasks.yaml"
}
