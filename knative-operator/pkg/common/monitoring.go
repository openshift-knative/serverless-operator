package common

import (
	"fmt"
	"golang.org/x/net/context"
	"os"

	mfclient "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// OpenshiftServerlessInstalledNamespace is the ns where knative serving operator has been installed
	OpenshiftServerlessInstalledNamespace = "NAMESPACE"
	// service monitor created successfully when monitoringLabel added to namespace
	monitoringLabel = "openshift.io/cluster-monitoring"
	rolePath        = "deploy/role_service_monitor.yaml"
)

func AddMonitoringLabelToNamespace(namespace string, api client.Client) error {
	ns := &v1.Namespace{}
	if err := api.Get(context.TODO(), client.ObjectKey{Name: namespace}, ns); err != nil {
		return err
	}
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	ns.Labels[monitoringLabel] = "true"
	if err := api.Update(context.TODO(), ns); err != nil {
		log.Error(err, fmt.Sprintf("could not add label %q to namespace %q", monitoringLabel, namespace))
		return err
	}
	return nil
}

func CreateRoleAndRoleBinding(instance *appsv1.Deployment, namespace, path string, client client.Client) error {
	manifest, err := mf.NewManifest(path, mf.UseClient(mfclient.NewClient(client)))
	if err != nil {
		log.Error(err, "Unable to create role and roleBinding ServiceMonitor install manifest")
		return err
	}
	instance.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	// this is typical probably not needed as uid is enough for ownership
	instance.SetNamespace(namespace)

	transforms := []mf.Transformer{mf.InjectOwner(instance)}
	if len(namespace) > 0 {
		transforms = append(transforms, mf.InjectNamespace(namespace))
	}
	transforms = append(transforms, updateRoleBindingSubResource)
	if manifest, err = manifest.Transform(transforms...); err != nil {
		log.Error(err, "Unable to transform role and roleBinding serviceMonitor manifest")
		return err
	}
	if err := manifest.Apply(); err != nil {
		log.Error(err, "Unable to create role and roleBinding for ServiceMonitor")
		return err
	}
	return nil
}

func GetOperatorNamespace() (string, error) {
	ns, found := os.LookupEnv(OpenshiftServerlessInstalledNamespace)
	if !found {
		return "", fmt.Errorf("%s must be set", OpenshiftServerlessInstalledNamespace)
	}
	return ns, nil
}

func SetUpMonitoringRequirements(api client.Client) error {
	ns, err := GetOperatorNamespace()
	if err != nil {
		return err
	}
	err = AddMonitoringLabelToNamespace(ns, api)
	if err != nil {
		return err
	}
	d, err := GetServerlessOperatorDeployment(api, ns)
	if err != nil {
		return err
	}
	err = CreateRoleAndRoleBinding(d, ns, rolePath, api)
	if err != nil {
		return err
	}
	return nil
}

func GetServerlessOperatorDeployment(api client.Client, namespace string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "knative-openshift", Namespace: namespace}
	err := api.Get(context.TODO(), key, deployment)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func updateRoleBindingSubResource(resource *unstructured.Unstructured) error {
	if resource.GetKind() != "RoleBinding" {
		return nil
	}
	var rb = &rbacv1.RoleBinding{}
	if err := scheme.Scheme.Convert(resource, rb, nil); err != nil {
		return err
	}
	sub := rb.Subjects[0]
	sub.Namespace = "openshift-monitoring"
	rb.Subjects = []rbacv1.Subject{sub}
	return scheme.Scheme.Convert(rb, resource, nil)
}
