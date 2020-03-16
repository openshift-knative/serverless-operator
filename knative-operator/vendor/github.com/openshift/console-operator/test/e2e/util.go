package e2e

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	consoleapi "github.com/openshift/console-operator/pkg/api"
	"github.com/openshift/console-operator/test/e2e/framework"
)

// Each of these tests helpers are similar, they only vary in the
// resource they are GETting and PATCHing.
// After the patch is done the test will poll the given resource.
// In case the console-operator is Managed state the patched data should
// not be equal to the one obtained after patch is applied.
// In case the console-operator is Unmanaged state the patched data should
// be equal to the one obtained after patch is applied.

var pollTimeout = 10 * time.Second

func patchAndCheckConfigMap(t *testing.T, client *framework.ClientSet, isOperatorManaged bool) error {
	t.Logf("patching Data on the console ConfigMap")
	configMap, err := client.Core.ConfigMaps(consoleapi.OpenShiftConsoleNamespace).Patch(consoleapi.OpenShiftConsoleConfigMapName, types.MergePatchType, []byte(`{"data": {"console-config.yaml": "test"}}`))
	if err != nil {
		return err
	}
	patchedData := configMap.Data

	t.Logf("polling for patched Data on the console ConfigMap")
	err = wait.Poll(1*time.Second, pollTimeout, func() (stop bool, err error) {
		configMap, err = framework.GetConsoleConfigMap(client)
		if err != nil {
			return true, err
		}
		newData := configMap.Data
		if isOperatorManaged {
			return !reflect.DeepEqual(patchedData, newData), nil
		}
		return reflect.DeepEqual(patchedData, newData), nil
	})
	return err
}

func patchAndCheckService(t *testing.T, client *framework.ClientSet, isOperatorManaged bool) error {
	t.Logf("patching Annotation on the console Service")
	service, err := client.Core.Services(consoleapi.OpenShiftConsoleNamespace).Patch(consoleapi.OpenShiftConsoleServiceName, types.MergePatchType, []byte(`{"metadata": {"annotations": {"service.alpha.openshift.io/serving-cert-secret-name": "test"}}}`))
	if err != nil {
		return err
	}
	patchedData := service.GetAnnotations()

	t.Logf("polling for patched Annotation on the console Service")
	err = wait.Poll(1*time.Second, pollTimeout, func() (stop bool, err error) {
		service, err = framework.GetConsoleService(client)
		if err != nil {
			return true, err
		}
		newData := service.GetAnnotations()
		if isOperatorManaged {
			return !reflect.DeepEqual(patchedData, newData), nil
		}
		return reflect.DeepEqual(patchedData, newData), nil
	})
	return err
}

func patchAndCheckRoute(t *testing.T, client *framework.ClientSet, isOperatorManaged bool) error {
	t.Logf("patching TargetPort on the console Route")
	route, err := client.Routes.Routes(consoleapi.OpenShiftConsoleNamespace).Patch(consoleapi.OpenShiftConsoleRouteName, types.MergePatchType, []byte(`{"spec": {"port": {"targetPort": "http"}}}`))
	if err != nil {
		return err
	}
	patchedData := route.Spec.Port.TargetPort

	t.Logf("polling for patched TargetPort on the console Route")
	err = wait.Poll(1*time.Second, pollTimeout, func() (stop bool, err error) {
		route, err = framework.GetConsoleRoute(client)
		if err != nil {
			return true, err
		}
		newData := route.Spec.Port.TargetPort
		if isOperatorManaged {
			return !reflect.DeepEqual(patchedData, newData), nil
		}
		return reflect.DeepEqual(patchedData, newData), nil
	})
	return err
}

func patchAndCheckConsoleCLIDownloads(t *testing.T, client *framework.ClientSet, isOperatorManaged bool, consoleCLIDownloadName string) error {
	t.Logf("patching DisplayName on the %s ConsoleCLIDownloads custom resource", consoleCLIDownloadName)
	consoleCLIDownload, err := client.ConsoleCliDownloads.Patch(consoleCLIDownloadName, types.MergePatchType, []byte(`{"spec": {"displayName": "test"}}`))
	if err != nil {
		return err
	}
	patchedData := consoleCLIDownload.Spec.DisplayName

	t.Logf("polling for patched DisplayName on the %s ConsoleCLIDownloads custom resource", consoleCLIDownloadName)
	err = wait.Poll(1*time.Second, pollTimeout, func() (stop bool, err error) {
		consoleCLIDownload, err = framework.GetConsoleCLIDownloads(client, consoleCLIDownloadName)
		if err != nil {
			return true, err
		}
		newData := consoleCLIDownload.Spec.DisplayName
		if isOperatorManaged {
			return !reflect.DeepEqual(patchedData, newData), nil
		}
		return reflect.DeepEqual(patchedData, newData), nil
	})
	return err
}
