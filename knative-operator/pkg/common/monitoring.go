package common

import (
	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"golang.org/x/net/context"
	"os"

	mfclient "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// operatorDeploymentNameEnvKey is the name of the deployment of the Openshift serverless operator
	operatorDeploymentNameEnvKey = "DEPLOYMENT_NAME"
	// service monitor created successfully when monitoringLabel added to namespace
	monitoringLabel = "openshift.io/cluster-monitoring"
	rolePath        = "deploy/role_service_monitor.yaml"
	testRolePath    = "TEST_ROLE_PATH"
)

func SetupMonitoringRequirements(monitoredNamespace string, api client.Client) error {
	err := addMonitoringLabelToNamespace(monitoredNamespace, api)
	if err != nil {
		return err
	}
	operatorDeploymentNamespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	d, err := getServerlessOperatorDeployment(api, operatorDeploymentNamespace)
	if err != nil {
		return err
	}
	err = createRoleAndRoleBinding(d, monitoredNamespace, getRolePath(), api)
	if err != nil {
		return err
	}
	return nil
}

func getRolePath() string {
	// meant for testing only
	ns, found := os.LookupEnv(testRolePath)
	if found {
		return ns
	} else {
		return rolePath
	}
}

func addMonitoringLabelToNamespace(namespace string, api client.Client) error {
	ns := &v1.Namespace{}
	if err := api.Get(context.TODO(), client.ObjectKey{Name: namespace}, ns); err != nil {
		return err
	}
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	ns.Labels[monitoringLabel] = "true"
	if err := api.Update(context.TODO(), ns); err != nil {
		return fmt.Errorf("could not add label %q to namespace %q: %w", monitoringLabel, namespace, err)
	}
	return nil
}

func getServerlessOperatorDeployment(api client.Client, namespace string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	deploymentName, err := getOperatorDeploymentName()
	if err != nil {
		return nil, err
	}
	key := types.NamespacedName{Name: deploymentName, Namespace: namespace}
	err = api.Get(context.TODO(), key, deployment)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func createRoleAndRoleBinding(instance *appsv1.Deployment, namespace, path string, client client.Client) error {
	manifest, err := mf.NewManifest(path, mf.UseClient(mfclient.NewClient(client)))
	if err != nil {
		return fmt.Errorf("unable to create role and roleBinding ServiceMonitor install manifest: %w", err)
	}
	instance.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	// this is typical probably not needed as uid is enough for ownership
	instance.SetNamespace(namespace)
	transforms := []mf.Transformer{mf.InjectOwner(instance), injectNameSpace(namespace)}
	if manifest, err = manifest.Transform(transforms...); err != nil {
		return fmt.Errorf("unable to transform role and roleBinding serviceMonitor manifest: %w", err)
	}
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("unable to create role and roleBinding for ServiceMonitor %w", err)
	}
	return nil
}

func getOperatorDeploymentName() (string, error) {
	ns, found := os.LookupEnv(operatorDeploymentNameEnvKey)
	if !found {
		return "", fmt.Errorf("the environment variable %q must be set", operatorDeploymentNameEnvKey)
	}
	return ns, nil
}

// Use a custom transformation otherwise if mf.InjectNameSpace was used
// it would wrongly update rolebinding subresource namespace as well
func injectNameSpace(namespace string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind := u.GetKind()
		if kind == "Role" || kind == "RoleBinding" {
			u.SetNamespace(namespace)
		}
		return nil
	}
}
