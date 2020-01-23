package consoleclidownload

import (
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"

	mf "github.com/jcrossley3/manifestival"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const knConsoleCLIDownload = "deploy/resources/console_cli_download_kn.yaml"

var log = common.Log.WithName("consoleclidownload")

// Create creates ConsoleCLIDownload for kn CLI download links
func Create(instance *servingv1alpha1.KnativeServing, apiclient client.Client) error {
	log.Info("Creating ConsoleCLIDownload CR for kn")
	manifest, err := mf.NewManifest(knConsoleCLIDownload, false, apiclient)
	if err != nil {
		return err
	}

	return manifest.ApplyAll()
}

// Delete deletes ConsoleCLIDownload for kn CLI download links
func Delete(instance *servingv1alpha1.KnativeServing, apiclient client.Client) error {
	log.Info("Deleting ConsoleCLIDownload CR for kn")
	manifest, err := mf.NewManifest(knConsoleCLIDownload, false, apiclient)
	if err != nil {
		return err
	}

	return manifest.DeleteAll()
}
