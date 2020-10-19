package consoleclidownload

import (
	"context"
	"fmt"
	v1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
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
	knConsoleCLIDownloadService = "kn-cli"
	deprecatedResourceName      = "kn-cli-downloads"
)

var log = common.Log.WithName("consoleclidownload")

// Apply installs kn ConsoleCLIDownload and its required resources
func Apply(instance *servingv1alpha1.KnativeServing, apiclient client.Client, scheme *runtime.Scheme) error {
	if !instance.Status.IsReady() {
		// Don't return error, wait silently until Serving instance is ready
		return nil
	}
	// Remove deprecated resources from previous version
	if err := deleteDeprecatedResources(instance, apiclient); err != nil {
		return err
	}
	service := &servingv1.Service{}
	if err := reconcileKnCCDResources(instance, apiclient, scheme, service); err != nil {
		return err
	}
	if !service.Status.IsReady() {
		return fmt.Errorf("Knative Service %q/%q not ready yet", knConsoleCLIDownloadService, instance.GetNamespace())
	}
	if err := reconcileKnConsoleCLIDownload(apiclient, instance, service); err != nil {
		return err
	}
	return nil
}

// reconcileKnCCDResources reconciles required resources viz Knative Service
// which will serve kn cross platform binaries within cluster
func reconcileKnCCDResources(instance *servingv1alpha1.KnativeServing, apiclient client.Client, scheme *runtime.Scheme, service *servingv1.Service) error {
	log.Info("Installing kn ConsoleCLIDownload resources")
	err := apiclient.Get(context.TODO(), client.ObjectKey{Namespace: instance.GetNamespace(), Name: knConsoleCLIDownloadService}, service)
	switch {
	case apierrors.IsNotFound(err):
		tmpService := makeKnService(os.Getenv("IMAGE_SERVING_KN_CLI_ARTIFACTS"), instance)
		if err := apiclient.Create(context.TODO(), tmpService); err != nil {
			return err
		}
	case err == nil:
		tmpService := makeKnService(os.Getenv("IMAGE_SERVING_KN_CLI_ARTIFACTS"), instance)
		serviceFromClusterDC := service.DeepCopy()
		if !equality.Semantic.DeepEqual(service.Spec, tmpService.Spec) {
			serviceFromClusterDC.Spec = tmpService.Spec
			if err := apiclient.Update(context.TODO(), serviceFromClusterDC); err != nil {
				return err
			}
		}
	default:
		return err
	}
	return nil
}

// reconcileKnConsoleCLIDownload reconciles kn ConsoleCLIDownload by finding
// kn download resource route URL and populating spec accordingly
func reconcileKnConsoleCLIDownload(apiclient client.Client, instance *servingv1alpha1.KnativeServing, knService *servingv1.Service) error {

	log.Info("Installing kn ConsoleCLIDownload")
	ctx := context.TODO()

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
	if err := apiclient.Delete(context.TODO(), makeKnService("", instance)); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete kn ConsoleCLIDownload Service: %w", err)
	}

	return nil
}

// deleteDeprecatedResources removes deprecated resources created by previous versions
func deleteDeprecatedResources(instance *servingv1alpha1.KnativeServing, apiclient client.Client) error {
	metaName := metav1.ObjectMeta{
		Name:      deprecatedResourceName,
		Namespace: instance.Namespace,
	}
	toDelete := []runtime.Object{
		&appsv1.Deployment{ObjectMeta: metaName},
		&corev1.Service{ObjectMeta: metaName},
		&v1.Route{ObjectMeta: metaName},
	}
	for _, obj := range toDelete {
		if err := apiclient.Delete(context.TODO(), obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete deprecated kn ConsoleCLIDownload %s: %w", obj.GetObjectKind().GroupVersionKind().Kind, err)
		}
	}
	return nil
}

// makeKnService makes Knative Service object and its SPEC from provided image parameter
func makeKnService(image string, instance *servingv1alpha1.KnativeServing) *servingv1.Service {
	// OwnerReference is not used to handle ksvc cleanup due to race condition with control-plane deletion.
	// In such a case route's finalizer blocks clean removal of resources.
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
