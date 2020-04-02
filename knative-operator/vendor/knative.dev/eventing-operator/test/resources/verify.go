/*
Copyright 2020 The Knative Authors
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

package resources

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	// Mysteriously required to support GCP auth (required by k8s libs).
	// Apparently just importing it is enough. @_@ side effects @_@.
	// https://github.com/kubernetes/client-go/issues/242
	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"knative.dev/eventing-operator/test"
	pkgTest "knative.dev/pkg/test"
)

// Setup creates the client objects needed in the e2e tests.
func Setup(t *testing.T) *test.Clients {
	clients, err := test.NewClients(
		pkgTest.Flags.Kubeconfig,
		pkgTest.Flags.Cluster)
	if err != nil {
		t.Fatalf("Couldn't initialize clients: %v", err)
	}
	return clients
}

// AssertKEOperatorCRReadyStatus verifies if the KnativeEventing can reach the READY status.
func AssertKEOperatorCRReadyStatus(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	if _, err := WaitForKnativeEventingState(clients.KnativeEventing(), names.KnativeEventing,
		IsKnativeEventingReady); err != nil {
		t.Fatalf("KnativeService %q failed to get to the READY status: %v", names.KnativeEventing, err)
	}
}

// DeleteAndVerifyDeployments verify whether all the deployments for knative eventing are able to recreate, when they are deleted.
func DeleteAndVerifyDeployments(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	dpList, err := clients.KubeClient.Kube.AppsV1().Deployments(names.Namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to get any deployment under the namespace %q: %v",
			test.EventingOperatorNamespace, err)
	}
	if len(dpList.Items) == 0 {
		t.Fatalf("No deployment under the namespace %q was found",
			test.EventingOperatorNamespace)
	}
	// Delete the first deployment and verify the operator recreates it
	deployment := dpList.Items[0]
	if err := clients.KubeClient.Kube.AppsV1().Deployments(deployment.Namespace).Delete(deployment.Name,
		&metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Failed to delete deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
	}

	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		dep, err := clients.KubeClient.Kube.AppsV1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			// If the deployment is not found, we continue to wait for the availability.
			if apierrs.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return IsDeploymentAvailable(dep)
	})

	if waitErr != nil {
		t.Fatalf("The deployment %s/%s failed to reach the desired state: %v", deployment.Namespace, deployment.Name, err)
	}

	if _, err := WaitForKnativeEventingState(clients.KnativeEventing(), test.EventingOperatorName,
		IsKnativeEventingReady); err != nil {
		t.Fatalf("KnativeService %q failed to reach the desired state: %v", test.EventingOperatorName, err)
	}
	t.Logf("The deployment %s/%s reached the desired state.", deployment.Namespace, deployment.Name)
}

// KEOperatorCRDelete deletes tha KnativeEventing to see if all resources will be deleted
func KEOperatorCRDelete(t *testing.T, clients *test.Clients, names test.ResourceNames) {
	if err := clients.KnativeEventing().Delete(names.KnativeEventing, &metav1.DeleteOptions{}); err != nil {
		t.Fatalf("KnativeEventing %q failed to delete: %v", names.KnativeEventing, err)
	}
	_, b, _, _ := runtime.Caller(0)
	m, err := mfc.NewManifest(filepath.Join((filepath.Dir(b)+"/.."), "config/"), clients.Config)
	if err != nil {
		t.Fatal("Failed to load manifest", err)
	}
	if err := verifyNoKnativeEventings(clients); err != nil {
		t.Fatal(err)
	}
	// verify all but the CRD's and the Namespace are gone
	for _, u := range m.Filter(mf.NotCRDs, mf.Complement(mf.ByKind("Namespace"))).Resources() {
		waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
			if _, err := m.Client.Get(&u); apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
		if waitErr != nil {
			t.Fatalf("The %s %s failed to be deleted: %v", u.GetKind(), u.GetName(), waitErr)
		}
	}
	// verify all the CRD's remain
	for _, u := range m.Filter(mf.JustCRDs).Resources() {
		if _, err := m.Client.Get(&u); apierrs.IsNotFound(err) {
			t.Fatalf("The %s CRD was deleted", u.GetName())
		}
	}
}

func verifyNoKnativeEventings(clients *test.Clients) error {
	eventings, err := clients.KnativeEventingAll().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(eventings.Items) > 0 {
		return errors.New("Unable to verify cluster-scoped resources are deleted if any KnativeEventing exists")
	}
	return nil
}
