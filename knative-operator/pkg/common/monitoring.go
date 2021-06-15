package common

import (
	"context"
	"fmt"
	"os"

	mfclient "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// operatorDeploymentNameEnvKey is the name of the deployment of the Openshift serverless operator
	operatorDeploymentNameEnvKey = "DEPLOYMENT_NAME"
	// service monitor created successfully when monitoringLabel added to namespace
	monitoringLabel = "openshift.io/cluster-monitoring"
)

func SetupMonitoringRequirements(api client.Client, instance mf.Owner) error {
	err := addMonitoringLabelToNamespace(instance.GetNamespace(), api)
	if err != nil {
		return err
	}
	err = createRoleAndRoleBinding(instance, instance.GetNamespace(), api)
	if err != nil {
		return err
	}
	return nil
}

func RemoveOldServiceMonitorResourcesIfExist(namespace string, api client.Client) error {
	oldSM := monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "knative-openshift-metrics",
		},
	}
	oldService := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      oldSM.Name,
		},
	}
	if err := api.Delete(context.Background(), &oldSM); err != nil && !errors.IsNotFound(err) {
		return err
	}
	if err := api.Delete(context.Background(), &oldService); err != nil && !errors.IsNotFound(err) {
		return err
	}
	// Delete old sms to avoid alerting for target being down, SRVKE-838
	oldSM.SetNamespace("knative-eventing")
	oldSM.SetName("knative-eventing-metrics-broker-filter")
	if err := api.Delete(context.Background(), &oldSM); err != nil && !errors.IsNotFound(err) {
		return err
	}
	oldSM.SetName("knative-eventing-metrics-broker-ingr")
	if err := api.Delete(context.Background(), &oldSM); err != nil && !errors.IsNotFound(err) {
		return err
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

func createRoleAndRoleBinding(instance mf.Owner, namespace string, client client.Client) error {
	clientOptions := mf.UseClient(mfclient.NewClient(client))
	rbacManifest, err := createRBACManifestForPrometheusAccount(namespace, clientOptions)
	if err != nil {
		return err
	}
	transforms := []mf.Transformer{mf.InjectOwner(instance)}
	if *rbacManifest, err = rbacManifest.Transform(transforms...); err != nil {
		return fmt.Errorf("unable to transform role and roleBinding manifest for Prometheus account: %w", err)
	}
	if err := rbacManifest.Apply(); err != nil {
		return fmt.Errorf("unable to create role and roleBinding for Prometheus account %w", err)
	}
	return nil
}

func createRBACManifestForPrometheusAccount(ns string, options mf.Option) (*mf.Manifest, error) {
	var roleU = &unstructured.Unstructured{}
	var rbU = &unstructured.Unstructured{}
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving-prometheus-k8s",
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"services", "endpoints", "pods"},
			Verbs:     []string{"get", "list", "watch"},
		}},
	}
	rb := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving-prometheus-k8s",
			Namespace: ns,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "prometheus-k8s",
			Namespace: "openshift-monitoring",
		}},
	}
	if err := scheme.Scheme.Convert(&role, roleU, nil); err != nil {
		return nil, err
	}
	if err := scheme.Scheme.Convert(&rb, rbU, nil); err != nil {
		return nil, err
	}
	rbacManifest, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{*roleU, *rbU}), options)
	if err != nil {
		return nil, err
	}
	return &rbacManifest, nil
}

func getOperatorDeploymentName() (string, error) {
	ns, found := os.LookupEnv(operatorDeploymentNameEnvKey)
	if !found {
		return "", fmt.Errorf("the environment variable %q must be set", operatorDeploymentNameEnvKey)
	}
	return ns, nil
}
