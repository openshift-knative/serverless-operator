package framework

import (
	"fmt"
	"testing"
	"time"

	yaml "gopkg.in/yaml.v2"

	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	consoleapi "github.com/openshift/console-operator/pkg/api"
)

var (
	// AsyncOperationTimeout is how long we want to wait for asynchronous
	// operations to complete. ForeverTestTimeout is not long enough to create
	// several replicas and get them available on a slow machine.
	// Setting this to 5 minutes:w

	AsyncOperationTimeout = 5 * time.Minute
)

type TestingResource struct {
	kind      string
	name      string
	namespace string
}

func getTestingResources() []TestingResource {
	return []TestingResource{
		{"ConfigMap", consoleapi.OpenShiftConsoleConfigMapName, consoleapi.OpenShiftConsoleNamespace},
		{"ConsoleCLIDownloads", consoleapi.OCCLIDownloadsCustomResourceName, ""},
		{"ConsoleCLIDownloads", consoleapi.ODOCLIDownloadsCustomResourceName, ""},
		{"Deployment", consoleapi.OpenShiftConsoleDeploymentName, consoleapi.OpenShiftConsoleNamespace},
		{"Route", consoleapi.OpenShiftConsoleRouteName, consoleapi.OpenShiftConsoleNamespace},
		{"Service", consoleapi.OpenShiftConsoleServiceName, consoleapi.OpenShiftConsoleNamespace},
	}
}

func SetClusterProxyConfig(proxyConfig configv1.ProxySpec, client *ClientSet) error {
	_, err := client.Proxy.Proxies().Patch(consoleapi.ConfigResourceName, types.MergePatchType, []byte(fmt.Sprintf(`{"spec": {"httpProxy": "%s", "httpsProxy": "%s", "noProxy": "%s"}}`, proxyConfig.HTTPProxy, proxyConfig.HTTPSProxy, proxyConfig.NoProxy)))
	return err
}

func ResetClusterProxyConfig(client *ClientSet) error {
	_, err := client.Proxy.Proxies().Patch(consoleapi.ConfigResourceName, types.MergePatchType, []byte(`{"spec": {"httpProxy": "", "httpsProxy": "", "noProxy": ""}}`))
	return err
}

func DeleteAll(t *testing.T, client *ClientSet) {
	resources := getTestingResources()

	for _, resource := range resources {
		t.Logf("deleting console's %s %s...", resource.name, resource.kind)
		if err := DeleteCompletely(
			func() (runtime.Object, error) {
				return GetResource(client, resource)
			},
			func(*metav1.DeleteOptions) error {
				return deleteResource(client, resource)
			},
		); err != nil {
			t.Fatalf("unable to delete console's %s %s: %s", resource.name, resource.kind, err)
		}
	}
}

func GetResource(client *ClientSet, resource TestingResource) (runtime.Object, error) {
	var res runtime.Object
	var err error

	switch resource.kind {
	case "ConfigMap":
		res, err = client.Core.ConfigMaps(resource.namespace).Get(resource.name, metav1.GetOptions{})
	case "Service":
		res, err = client.Core.Services(resource.namespace).Get(resource.name, metav1.GetOptions{})
	case "Route":
		res, err = client.Routes.Routes(resource.namespace).Get(resource.name, metav1.GetOptions{})
	case "ConsoleCLIDownloads":
		res, err = client.ConsoleCliDownloads.Get(resource.name, metav1.GetOptions{})
	case "Deployment":
		res, err = client.Apps.Deployments(resource.namespace).Get(resource.name, metav1.GetOptions{})
	default:
		err = fmt.Errorf("error getting resource: resource %s not identified", resource.kind)
	}
	return res, err
}

// custom-logo in openshift-console should exist when custom branding is used
func GetCustomLogoConfigMap(client *ClientSet) (*corev1.ConfigMap, error) {
	return client.Core.ConfigMaps(consoleapi.OpenShiftConsoleNamespace).Get(consoleapi.OpenShiftCustomLogoConfigMapName, metav1.GetOptions{})
}

func GetConsoleConfigMap(client *ClientSet) (*corev1.ConfigMap, error) {
	return client.Core.ConfigMaps(consoleapi.OpenShiftConsoleNamespace).Get(consoleapi.OpenShiftConsoleConfigMapName, metav1.GetOptions{})
}

func GetConsoleService(client *ClientSet) (*corev1.Service, error) {
	return client.Core.Services(consoleapi.OpenShiftConsoleNamespace).Get(consoleapi.OpenShiftConsoleServiceName, metav1.GetOptions{})
}

func GetConsoleRoute(client *ClientSet) (*routev1.Route, error) {
	return client.Routes.Routes(consoleapi.OpenShiftConsoleNamespace).Get(consoleapi.OpenShiftConsoleRouteName, metav1.GetOptions{})
}

func GetConsoleDeployment(client *ClientSet) (*appv1.Deployment, error) {
	return client.Apps.Deployments(consoleapi.OpenShiftConsoleNamespace).Get(consoleapi.OpenShiftConsoleDeploymentName, metav1.GetOptions{})
}

func GetConsoleCLIDownloads(client *ClientSet, consoleCLIDownloadName string) (*consolev1.ConsoleCLIDownload, error) {
	return client.ConsoleCliDownloads.Get(consoleCLIDownloadName, metav1.GetOptions{})
}

func deleteResource(client *ClientSet, resource TestingResource) error {
	var err error
	switch resource.kind {
	case "ConfigMap":
		err = client.Core.ConfigMaps(resource.namespace).Delete(resource.name, &metav1.DeleteOptions{})
	case "Service":
		err = client.Core.Services(resource.namespace).Delete(resource.name, &metav1.DeleteOptions{})
	case "Route":
		err = client.Routes.Routes(resource.namespace).Delete(resource.name, &metav1.DeleteOptions{})
	case "ConsoleCLIDownloads":
		err = client.ConsoleCliDownloads.Delete(resource.name, &metav1.DeleteOptions{})
	case "Deployment":
		err = client.Apps.Deployments(resource.namespace).Delete(resource.name, &metav1.DeleteOptions{})
	default:
		err = fmt.Errorf("error deleting resource: resource %s not identified", resource.kind)
	}
	return err
}

// DeleteCompletely sends a delete request and waits until the resource and
// its dependents are deleted.
func DeleteCompletely(getObject func() (runtime.Object, error), deleteObject func(*metav1.DeleteOptions) error) error {
	obj, err := getObject()
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	accessor, err := meta.Accessor(obj)
	uid := accessor.GetUID()

	policy := metav1.DeletePropagationForeground
	if err := deleteObject(&metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{
			UID: &uid,
		},
		PropagationPolicy: &policy,
	}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return wait.Poll(1*time.Second, AsyncOperationTimeout, func() (stop bool, err error) {
		obj, err = getObject()
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}

		accessor, err := meta.Accessor(obj)

		return accessor.GetUID() != uid, nil
	})
}

func ConsoleResourcesAvailable(client *ClientSet) error {
	resources := getTestingResources()
	// We have to test the `console-public` configmap in the TestManaged as well.
	resources = append(resources, TestingResource{"ConfigMap", consoleapi.OpenShiftConsolePublicConfigMapName, consoleapi.OpenShiftConfigManagedNamespace})

	errChan := make(chan error)
	for _, resource := range resources {
		go IsResourceAvailable(errChan, client, resource)
	}

	checkErr := <-errChan
	return checkErr
}

// IsResourceAvailable checks if tested resource is available during a 30 second period.
// if the resource does not exist by the end of the period, an error will be returned.
func IsResourceAvailable(errChan chan error, client *ClientSet, resource TestingResource) {
	counter := 0
	maxCount := 30
	err := wait.Poll(1*time.Second, AsyncOperationTimeout, func() (stop bool, err error) {
		_, err = GetResource(client, resource)
		if err == nil {
			return true, nil
		}
		if counter == maxCount {
			if err != nil {
				return true, fmt.Errorf("deleted console %s %s was not recreated", resource.kind, resource.name)
			}
			return true, nil
		}
		counter++
		return false, nil
	})
	errChan <- err
}

// checks 3 times if the resources are unavailable
// - is fine if fails or 1st or 2nd run, resources could be in the process of being removed
// - is not fine if resources disappear, then reappear
// - it seems to take a bit longer to remove resources, so this wrapper should account for that.
func ConsoleResourcesUnavailable(client *ClientSet) error {
	var failed error = nil
	// give it 3 tries, then fail
	for i := 0; i < 3; i++ {
		// testing resources are hard-coded in this func.
		err := LoopResources(client, IsResourceUnavailable)
		fmt.Printf("validating console resources have been removed... %v\n", err == nil)
		failed = err
	}
	return failed
}

func LoopResources(client *ClientSet, inner func(errChan chan error, client *ClientSet, resource TestingResource)) error {
	resources := getTestingResources()

	errChan := make(chan error)
	for _, resource := range resources {
		go inner(errChan, client, resource)
	}
	checkErr := <-errChan

	return checkErr
}

// IsResourceUnavailable checks if tested resource is unavailable during a 15 second period.
// If the resource exists during that time, an error will be returned.
func IsResourceUnavailable(errChan chan error, client *ClientSet, resource TestingResource) {
	counter := 0
	maxCount := 15
	err := wait.Poll(1*time.Second, AsyncOperationTimeout, func() (stop bool, err error) {

		obtainedResource, err := GetResource(client, resource)
		if err == nil {

			yamlBytes, err := yaml.Marshal(obtainedResource)
			if err != nil {
				fmt.Printf("error marshalling yaml for %s %s %v", resource.kind, resource.name, err)
			}

			return true, fmt.Errorf("deleted console %s %s was recreated: %#v", resource.kind, resource.name, string(yamlBytes))
		}
		if !errors.IsNotFound(err) {
			return true, err
		}
		counter++
		if counter == maxCount {
			return true, nil
		}
		return false, nil
	})
	errChan <- err
}
