package consoleutil

import (
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/consoleclidownload"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/quickstart"
)

func SetConsoleCRDInstalled(crd string) {
	switch crd {
	case consoleclidownload.CLIDownloadCRDName:
		consoleclidownload.ConsoleCLIDownloadsCRDInstalled.Store(true)
	case quickstart.QuickStartsCRDName:
		quickstart.ConsoleQuickStartsCRDInstalled.Store(true)
	}
}

func RequiredConsoleCRDMissing() bool {
	return !consoleclidownload.ConsoleCLIDownloadsCRDInstalled.Load() || !quickstart.ConsoleQuickStartsCRDInstalled.Load()
}

func AnyRequiredConsoleCRDAvailable() bool {
	return consoleclidownload.ConsoleCLIDownloadsCRDInstalled.Load() || quickstart.ConsoleQuickStartsCRDInstalled.Load()
}
