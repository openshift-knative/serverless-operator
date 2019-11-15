/*
Copyright 2019 The Knative Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	mf "github.com/jcrossley3/manifestival"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/test/logstream"
	"knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"knative.dev/serving-operator/test"
	"knative.dev/serving-operator/test/resources"
)

func testKnativeServingDeployment(ctx *test.Context) {
	t := ctx.T()
	r := ctx.Runner()
	cancel := logstream.Start(t)
	defer cancel()
	clients := Setup(t)

	names := test.ResourceNames{
		KnativeServing: test.ServingOperatorName,
		Namespace:      test.ServingOperatorNamespace,
	}

	test.CleanupOnInterrupt(func() { test.TearDown(clients, names) })
	defer test.TearDown(clients, names)

	// Create a KnativeServing
	if _, err := resources.CreateKnativeServing(clients.KnativeServing(), names); err != nil {
		t.Fatalf("KnativeService %q failed to create: %v", names.KnativeServing, err)
	}

	// Test if KnativeServing can reach the READY status
	r.Run("create", func(t *testing.T) {
		knativeServingVerify(t, clients, names)
	})

	r.Run("configure", func(t *testing.T) {
		knativeServingVerify(t, clients, names)
		knativeServingConfigure(t, clients, names)
	})

	// Delete the deployments one by one to see if they will be recreated.
	r.Run("restore", func(t *testing.T) {
		knativeServingVerify(t, clients, names)
		deploymentRecreation(t, clients, names)
	})

	// Delete the KnativeServing to see if all the deployments will be removed as well
	r.Run("delete", func(t *testing.T) {
		knativeServingVerify(t, clients, names)
		knativeServingDeletion(t, clients, names)
		verifyClusterResourceDeletion(t, clients)
	})
}

// knativeServingVerify verifies if the KnativeServing can reach the READY status.
func knativeServingVerify(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	if _, err := resources.WaitForKnativeServingState(clients.KnativeServing(), names.KnativeServing,
		resources.IsKnativeServingReady); err != nil {
		t.Fatalf("KnativeService %q failed to get to the READY status: %v", names.KnativeServing, err)
	}

}

// knativeServingConfigure verifies that KnativeServing config is set properly
func knativeServingConfigure(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	// We'll arbitrarily choose the logging config
	configKey := "logging"
	configMapName := fmt.Sprintf("%s/config-%s", names.Namespace, configKey)
	// Get the existing KS without any spec
	ks, err := clients.KnativeServing().Get(names.KnativeServing, metav1.GetOptions{})
	// Add config to its spec
	ks.Spec = v1alpha1.KnativeServingSpec{
		Config: map[string]map[string]string{
			configKey: {
				"loglevel.controller": "debug",
			},
		},
	}
	// Update it
	if ks, err = clients.KnativeServing().Update(ks); err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}
	// Verifty the relevant configmap has been updated
	err = resources.WaitForConfigMap(configMapName, clients.KubeClient.Kube, func(m map[string]string) bool {
		return m["loglevel.controller"] == "debug"
	})
	if err != nil {
		t.Fatal("The operator failed to update the configmap")
	}
	// Now remove the config from the spec and update
	ks.Spec = v1alpha1.KnativeServingSpec{}
	if ks, err = clients.KnativeServing().Update(ks); err != nil {
		t.Fatalf("KnativeServing %q failed to update: %v", names.KnativeServing, err)
	}
	// And verify the configmap entry is gone
	err = resources.WaitForConfigMap(configMapName, clients.KubeClient.Kube, func(m map[string]string) bool {
		_, exists := m["loglevel.controller"]
		return !exists
	})
	if err != nil {
		t.Fatal("The operator failed to revert the configmap")
	}
}

// deploymentRecreation verify whether all the deployments for knative serving are able to recreate, when they are deleted.
func deploymentRecreation(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	dpList, err := clients.KubeClient.Kube.AppsV1().Deployments(names.Namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to get any deployment under the namespace %q: %v",
			test.ServingOperatorNamespace, err)
	}
	if len(dpList.Items) == 0 {
		t.Fatalf("No deployment under the namespace %q was found",
			test.ServingOperatorNamespace)
	}
	// Delete the first deployment and verify the operator recreates it
	deployment := dpList.Items[0]
	if err := clients.KubeClient.Kube.AppsV1().Deployments(deployment.Namespace).Delete(deployment.Name,
		&metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Failed to delete deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
	}

	waitErr := wait.PollImmediate(resources.Interval, resources.Timeout, func() (bool, error) {
		dep, err := clients.KubeClient.Kube.AppsV1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			// If the deployment is not found, we continue to wait for the availability.
			if apierrs.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return resources.IsDeploymentAvailable(dep)
	})

	if waitErr != nil {
		t.Fatalf("The deployment %s/%s failed to reach the desired state: %v", deployment.Namespace, deployment.Name, err)
	}

	if _, err := resources.WaitForKnativeServingState(clients.KnativeServing(), test.ServingOperatorName,
		resources.IsKnativeServingReady); err != nil {
		t.Fatalf("KnativeService %q failed to reach the desired state: %v", test.ServingOperatorName, err)
	}
	t.Logf("The deployment %s/%s reached the desired state.", deployment.Namespace, deployment.Name)
}

// knativeServingDeletion deletes tha KnativeServing to see if all the deployments will be removed.
func knativeServingDeletion(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	if err := clients.KnativeServing().Delete(names.KnativeServing, &metav1.DeleteOptions{}); err != nil {
		t.Fatalf("KnativeService %q failed to delete: %v", names.KnativeServing, err)
	}

	dpList, err := clients.KubeClient.Kube.AppsV1().Deployments(names.Namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Error getting any deployment under the namespace %q: %v", names.Namespace, err)
	}

	for _, deployment := range dpList.Items {
		waitErr := wait.PollImmediate(resources.Interval, resources.Timeout, func() (bool, error) {
			if _, err := clients.KubeClient.Kube.AppsV1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{}); err != nil {
				if apierrs.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})

		if waitErr != nil {
			t.Fatalf("The deployment %s/%s failed to be deleted: %v", deployment.Namespace, deployment.Name, waitErr)
		}
		t.Logf("The deployment %s/%s has been deleted.", deployment.Namespace, deployment.Name)
	}

	waitForNoKnativeServings(t, clients)
}

func verifyClusterResourceDeletion(t *testing.T, clients *test.Clients) {
	_, b, _, _ := runtime.Caller(0)
	m, err := mf.NewManifest(filepath.Join((filepath.Dir(b)+"/.."), "config/"), false, clients.Config)
	if err != nil {
		t.Fatal("Failed to load manifest", err)
	}
	if err := verifyNoKnativeServings(clients); err != nil {
		t.Fatal(err)
	}
	for _, u := range m.Resources {
		if u.GetNamespace() == "" && u.GetKind() != "Namespace" {
			waitErr := wait.PollImmediate(resources.Interval, resources.Timeout, func() (bool, error) {
				gvrs, _ := meta.UnsafeGuessKindToResource(u.GroupVersionKind())
				if _, err := clients.Dynamic.Resource(gvrs).Get(u.GetName(), metav1.GetOptions{}); apierrs.IsNotFound(err) {
					return true, nil
				} else {
					return false, err
				}
			})

			if waitErr != nil {
				t.Fatalf("The %s %s failed to be deleted: %v", u.GetKind(), u.GetName(), waitErr)
			}
			t.Logf("The %s %s has been deleted.", u.GetKind(), u.GetName())
		}
	}
}

func waitForNoKnativeServings(t *testing.T, clients *test.Clients) {
	t.Log("Waiting for knative-serving cr(s) to not be present")
	waitErr := wait.PollImmediate(resources.Interval, resources.Timeout, func() (bool, error) {
		list, err := clients.KnativeServingAll().List(metav1.ListOptions{})
		if apierrs.IsNotFound(err) || len(list.Items) == 0 {
			return true, nil
		}
		return false, err
	})
	if waitErr != nil {
		t.Fatalf("At least one knative-serving cr is still present after timeout was reached: %v", waitErr)
	}
	t.Log("No knative-serving cr is present")
}

func verifyNoKnativeServings(clients *test.Clients) error {
	servings, err := clients.KnativeServingAll().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(servings.Items) > 0 {
		return errors.New("Unable to verify cluster-scoped resources are deleted if any KnativeServing exists")
	}
	return nil
}
