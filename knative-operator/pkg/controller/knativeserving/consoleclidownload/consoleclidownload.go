package consoleclidownload

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"

	consolev1 "github.com/openshift/api/console/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	knCLIDownload               = "kn"
	knDownloadServer            = "kn-download-server"
	knConsoleCLIDownloadService = "kn-cli-downloads"
)

var log = common.Log.WithName("consoleclidownload")

// Apply installs kn ConsoleCLIDownload and its required resources
func Apply(instance *servingv1alpha1.KnativeServing, apiclient client.Client, scheme *runtime.Scheme) error {
	if err := reconcileKnCCDResources(instance, apiclient, scheme); err != nil {
		return err
	}

	if err := reconcileKnConsoleCLIDownload(apiclient, instance); err != nil {
		return err
	}

	return nil
}

// reconcileKnCCDResources reconciles required resources viz Knative Service
// which will serve kn cross platform binaries within cluster
func reconcileKnCCDResources(instance *servingv1alpha1.KnativeServing, apiclient client.Client, scheme *runtime.Scheme) error {
	log.Info("Installing kn ConsoleCLIDownload resources")
	serviceFromCluster := &servingv1.Service{}
	service := populateKnService(os.Getenv("IMAGE_KN_CLI_ARTIFACTS"), instance)
	err := apiclient.Get(context.TODO(), client.ObjectKey{Namespace: instance.GetNamespace(), Name: knConsoleCLIDownloadService}, serviceFromCluster)
	switch {
	case apierrors.IsNotFound(err):
		if err := apiclient.Create(context.TODO(), service); err != nil {
			return err
		}
	case err == nil:
		serviceFromClusterDC := serviceFromCluster.DeepCopy()
		changed := false
		if serviceFromCluster.Spec.Template.Spec.GetContainer().Image != service.Spec.Template.Spec.GetContainer().Image {
			serviceFromClusterDC.Spec.Template.Spec.GetContainer().Image = service.Spec.Template.Spec.GetContainer().Image
			changed = true
		}
		if !equality.Semantic.DeepEqual(serviceFromCluster.Spec.Template.Spec.GetContainer().Resources, service.Spec.Template.Spec.GetContainer().Resources) {
			serviceFromClusterDC.Spec.Template.Spec.GetContainer().Resources = service.Spec.Template.Spec.GetContainer().Resources
			changed = true
		}
		if changed {
			if err := apiclient.Update(context.TODO(), serviceFromClusterDC); err != nil {
				return err
			}
		}
	default:
		return err
	}

	if err := checkResources(instance, apiclient); err != nil {
		return err
	}

	return nil
}

// Check for Knative Service and URL of kn ConsoleCLIDownload resources
func checkResources(instance *servingv1alpha1.KnativeServing, api client.Client) error {
	log.Info("Checking deployments")
	service := &servingv1.Service{}
	if err := api.Get(context.TODO(), client.ObjectKey{Namespace: instance.GetNamespace(), Name: knConsoleCLIDownloadService}, service); err != nil {
		return err
	}
	if !service.Status.IsReady() {
		return fmt.Errorf("Knative Service %q/%q not ready yet", knConsoleCLIDownloadService, instance.GetNamespace())
	}
	if service.Status.URL == nil || service.Status.URL.String() == "" {
		return fmt.Errorf("Knative Service URL %q/%q not present yet", knConsoleCLIDownloadService, instance.GetNamespace())
	}
	return nil
}

// reconcileKnConsoleCLIDownload reconciles kn ConsoleCLIDownload by finding
// kn download resource route URL and populating spec accordingly
func reconcileKnConsoleCLIDownload(apiclient client.Client, instance *servingv1alpha1.KnativeServing) error {

	log.Info("Installing kn ConsoleCLIDownload")
	ctx := context.TODO()

	// find the Knative Service first
	knService := &servingv1.Service{}
	if err := apiclient.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: knConsoleCLIDownloadService}, knService); err != nil {
		return fmt.Errorf("failed to find kn ConsoleCLIDownload Knative Service: %v", err)
	}

	knRouteURL := knService.Status.URL
	if knRouteURL == nil || knRouteURL.String() == "" {
		return fmt.Errorf("failed to get kn ConsoleCLIDownload Knative Service URL")
	}

	knCCDGet := &consolev1.ConsoleCLIDownload{}
	knConsoleObj := populateKnConsoleCLIDownload(https(knRouteURL.Host), instance)

	// Check if kn ConsoleCLIDownload exists
	err := apiclient.Get(ctx, client.ObjectKey{Namespace: "", Name: knCLIDownload}, knCCDGet)
	switch {
	case apierrors.IsNotFound(err):
		if err := apiclient.Create(ctx, knConsoleObj); err != nil {
			return err
		}
	case err == nil:
		knCCDCopy := knCCDGet.DeepCopy()
		change := false
		if !equality.Semantic.DeepEqual(knCCDGet.Annotations, knConsoleObj.Annotations) {
			change = true
			knCCDCopy.Annotations = knConsoleObj.Annotations
		}
		if !equality.Semantic.DeepEqual(knCCDGet.Spec, knConsoleObj.Spec) {
			change = true
			knCCDCopy.Spec = knConsoleObj.Spec
		}
		// Update only if there's is a change
		if !change {
			return nil
		}
		log.Info("Updating kn ConsoleCLIDownload..")
		if err := apiclient.Update(ctx, knCCDCopy); err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

// Delete deletes kn ConsoleCLIDownload CO and respective deployment resources
func Delete(instance *servingv1alpha1.KnativeServing, apiclient client.Client, scheme *runtime.Scheme) error {
	log.Info("Deleting kn ConsoleCLIDownload CO")
	if err := apiclient.Delete(context.TODO(), populateKnConsoleCLIDownload("", nil)); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete kn ConsoleCLIDownload CO: %w", err)
	}

	log.Info("Deleting kn ConsoleCLIDownload Service")
	if err := apiclient.Delete(context.TODO(), populateKnService("", instance)); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete kn ConsoleCLIDownload Service: %w", err)
	}

	return nil
}

// populateKnService populates Knatie Service object and its SPEC with provided image
func populateKnService(image string, instance *servingv1alpha1.KnativeServing) *servingv1.Service {
	anno := make(map[string]string)
	if instance != nil {
		anno = map[string]string{
			common.ServingOwnerName:      instance.Name,
			common.ServingOwnerNamespace: instance.Namespace,
		}
	}
	service := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        knConsoleCLIDownloadService,
			Namespace:   instance.Namespace,
			Annotations: anno,
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: image,
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("10m"),
											corev1.ResourceMemory: resource.MustParse("50Mi"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return service
}

// populateKnConsoleCLIDownload populates kn ConsoleCLIDownload object and its SPEC
// using route's baseURL
func populateKnConsoleCLIDownload(baseURL string, instance *servingv1alpha1.KnativeServing) *consolev1.ConsoleCLIDownload {
	anno := make(map[string]string)
	if instance != nil {
		anno = map[string]string{
			common.ServingOwnerName:      instance.Name,
			common.ServingOwnerNamespace: instance.Namespace,
		}
	}
	return &consolev1.ConsoleCLIDownload{
		ObjectMeta: metav1.ObjectMeta{
			Name:        knCLIDownload,
			Annotations: anno,
		},
		Spec: consolev1.ConsoleCLIDownloadSpec{
			DisplayName: "kn - OpenShift Serverless Command Line Interface (CLI)",
			Description: "The OpenShift Serverless client `kn` is a CLI tool that allows you to fully manage OpenShift Serverless Serving and Eventing resources without writing a single line of YAML.",
			Links: []consolev1.Link{
				consolev1.Link{
					Text: "Download kn for Linux",
					Href: baseURL + "/amd64/linux/kn-linux-amd64.tar.gz",
				},
				consolev1.Link{
					Text: "Download kn for macOS",
					Href: baseURL + "/amd64/macos/kn-macos-amd64.tar.gz",
				},
				consolev1.Link{
					Text: "Download kn for Windows",
					Href: baseURL + "/amd64/windows/kn-windows-amd64.zip",
				},
			},
		},
	}
}

// copied from github.com/openshift/console-operator/pkg/console/subresource/util/util.go and modified
func https(host string) string {
	if host == "" {
		return ""
	}
	protocol := "https://"
	if strings.HasPrefix(host, protocol) {
		return host
	}
	return protocol + host
}
