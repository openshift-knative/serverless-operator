package common

import (
	"context"
	"fmt"
	"os"
	"strings"

	mfclient "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// operatorDeploymentNameEnvKey is the name of the deployment of the Openshift serverless operator
	operatorDeploymentNameEnvKey = "DEPLOYMENT_NAME"
	// service monitor created successfully when monitoringLabel added to namespace
	monitoringLabel                   = "openshift.io/cluster-monitoring"
	rolePath                          = "deploy/resources/monitoring/role-service-monitor.yaml"
	TestRolePath                      = "TEST_ROLE_PATH"
	operatorServiceMonitorNameEnvKey  = "OPERATOR_SERVICE_MONITOR_NAME"
	operatorServiceMonitorDefaultName = "knative-openshift-metrics-3"
)

func SetupMonitoringRequirements(api client.Client, instance mf.Owner) error {
	err := addMonitoringLabelToNamespace(instance.GetNamespace(), api)
	if err != nil {
		return err
	}
	err = createRoleAndRoleBinding(instance, instance.GetNamespace(), getRolePath(), api)
	if err != nil {
		return err
	}
	return nil
}

func RemoveOldServiceMonitorResources(namespace string, api client.Client) error {
	currentSMName := os.Getenv(operatorServiceMonitorNameEnvKey)
	if currentSMName == "" {
		currentSMName = operatorServiceMonitorDefaultName
	}
	smList := monitoringv1.ServiceMonitorList{}
	if err := api.List(context.TODO(), &smList, client.InNamespace(namespace)); err != nil {
		return err
	}
	for _, sm := range smList.Items {
		// Skip the sm that is being installed with the current operator version
		if sm.Name != currentSMName && strings.HasPrefix(sm.Name, "knative-openshift-metrics-") {
			// Delete service monitor and the related service monitor service, skip error if not found
			if err := api.Delete(context.TODO(), sm); err != nil && !errors.IsNotFound(err) {
				return err
			}
			service := v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      sm.Name,
				},
			}
			if err := api.Delete(context.TODO(), &service); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func GetServerlessOperatorDeployment(api client.Client, namespace string) (*appsv1.Deployment, error) {
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
	// Set version and kind here because it is required for the owner references
	// used by the role creation later on
	// currently k8s api returns no value for these fields, for more check:
	// https://github.com/kubernetes/client-go/issues/861
	deployment.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	return deployment, nil
}

func getRolePath() string {
	// meant for testing only
	ns, found := os.LookupEnv(TestRolePath)
	if found {
		return ns
	}
	return rolePath
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

func createRoleAndRoleBinding(instance mf.Owner, namespace, path string, client client.Client) error {
	manifest, err := mf.NewManifest(path, mf.UseClient(mfclient.NewClient(client)))
	if err != nil {
		return fmt.Errorf("unable to create role and roleBinding ServiceMonitor install manifest: %w", err)
	}
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
