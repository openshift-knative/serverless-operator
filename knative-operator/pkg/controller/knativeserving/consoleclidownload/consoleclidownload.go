package consoleclidownload

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const knCLIDownload = "kn"

var (
	operatorNamespace = os.Getenv(common.NamespaceEnvKey)
	log               = common.Log.WithName("consoleclidownload")
)

// Apply installs kn ConsoleCLIDownload and its required resources
func Apply(instance *operatorv1beta1.KnativeServing, apiclient client.Client, scheme *runtime.Scheme) error {
	route, err := reconcileKnConsoleCLIDownloadRoute(apiclient, instance)
	if err != nil {
		return err
	}

	return reconcileKnConsoleCLIDownload(apiclient, instance, route)
}

func reconcileKnConsoleCLIDownloadRoute(apiclient client.Client, instance *operatorv1beta1.KnativeServing) (*routev1.Route, error) {
	log.Info("Installing kn ConsoleCLIDownload Route")
	ctx := context.Background()

	route := &routev1.Route{}
	err := apiclient.Get(ctx, client.ObjectKey{Namespace: operatorNamespace, Name: knCLIDownload}, route)
	if apierrors.IsNotFound(err) {
		route = makeRoute(instance)
		if err := apiclient.Create(ctx, route); err != nil {
			return nil, fmt.Errorf("failed to create route for ConsoleCLIDownload: %w", err)
		}
		return route, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch route for ConsoleCLIDownload: %w", err)
	}

	newRoute := makeRoute(instance)
	if equality.Semantic.DeepEqual(route.Spec, newRoute.Spec) {
		// Equal, nothing to do here.
		return route, nil
	}

	route = route.DeepCopy()
	route.Spec = newRoute.Spec
	if err := apiclient.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to update route for ConsoleCLIDownload: %w", err)
	}
	return route, nil
}

// reconcileKnConsoleCLIDownload reconciles kn ConsoleCLIDownload by finding
// kn download resource route URL and populating spec accordingly
func reconcileKnConsoleCLIDownload(apiclient client.Client, instance *operatorv1beta1.KnativeServing, route *routev1.Route) error {
	log.Info("Installing kn ConsoleCLIDownload")
	ctx := context.TODO()

	knCCDGet := &consolev1.ConsoleCLIDownload{}
	knConsoleObj := populateKnConsoleCLIDownload(https(route.Spec.Host), instance)

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
func Delete(instance *operatorv1beta1.KnativeServing, apiclient client.Client, scheme *runtime.Scheme) error {
	log.Info("Deleting kn ConsoleCLIDownload CO")
	if err := apiclient.Delete(context.TODO(), populateKnConsoleCLIDownload("", instance)); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete kn ConsoleCLIDownload CO: %w", err)
	}

	log.Info("Deleting kn ConsoleCLIDownload Route")
	if err := apiclient.Delete(context.TODO(), makeRoute(instance)); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete kn ConsoleCLIDownload Service: %w", err)
	}

	return nil
}

func makeRoute(instance *operatorv1beta1.KnativeServing) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      knCLIDownload,
			Namespace: operatorNamespace,
			Annotations: map[string]string{
				socommon.ServingOwnerName:      instance.GetName(),
				socommon.ServingOwnerNamespace: instance.GetNamespace(),
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "knative-openshift-metrics-3",
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http-cli"),
			},
		},
	}
}

// populateKnConsoleCLIDownload populates kn ConsoleCLIDownload object and its SPEC
// using route's baseURL
func populateKnConsoleCLIDownload(baseURL string, instance *operatorv1beta1.KnativeServing) *consolev1.ConsoleCLIDownload {
	return &consolev1.ConsoleCLIDownload{
		ObjectMeta: metav1.ObjectMeta{
			Name: knCLIDownload,
			Annotations: map[string]string{
				socommon.ServingOwnerName:      instance.GetName(),
				socommon.ServingOwnerNamespace: instance.GetNamespace(),
			},
		},
		Spec: consolev1.ConsoleCLIDownloadSpec{
			DisplayName: "kn - OpenShift Serverless Command Line Interface (CLI)",
			Description: "The OpenShift Serverless client `kn` is a CLI tool that allows you to fully manage OpenShift Serverless Serving, Eventing, and Function resources without writing a single line of YAML.",
			Links: []consolev1.CLIDownloadLink{{
				Text: "Download kn for Linux for x86_64",
				Href: baseURL + "/kn-linux-amd64.tar.gz",
			}, {
				Text: "Download kn for Linux for IBM Power little endian",
				Href: baseURL + "/kn-linux-ppc64le.tar.gz",
			}, {
				Text: "Download kn for Linux for IBM Z",
				Href: baseURL + "/kn-linux-s390x.tar.gz",
			}, {
				Text: "Download kn for macOS",
				Href: baseURL + "/kn-macos-amd64.tar.gz",
			}, {
				Text: "Download kn for Windows",
				Href: baseURL + "/kn-windows-amd64.zip",
			}},
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
