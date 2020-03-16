package clidownloads

import (
	// standard lib
	"fmt"
	"time"

	// kube
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	// openshift
	v1 "github.com/openshift/api/console/v1"
	operatorsv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	"github.com/openshift/library-go/pkg/operator/v1helpers"

	// informers
	consoleinformersv1 "github.com/openshift/client-go/console/informers/externalversions/console/v1"
	operatorinformersv1 "github.com/openshift/client-go/operator/informers/externalversions/operator/v1"
	routesinformersv1 "github.com/openshift/client-go/route/informers/externalversions/route/v1"

	// clients
	consoleclientv1 "github.com/openshift/client-go/console/clientset/versioned/typed/console/v1"
	operatorclientv1 "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	routeclientv1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	// operator
	"github.com/openshift/console-operator/pkg/api"
	"github.com/openshift/console-operator/pkg/console/status"
	routesub "github.com/openshift/console-operator/pkg/console/subresource/route"
	"github.com/openshift/console-operator/pkg/console/subresource/util"
)

const (
	controllerWorkQueueKey = "clidownloads-sync-work-queue-key"
	controllerName         = "ConsoleCLIDownloadsSyncController"
)

type CLIDownloadsSyncController struct {
	// clients
	consoleCliDownloadsClient consoleclientv1.ConsoleCLIDownloadInterface
	routeClient               routeclientv1.RoutesGetter
	operatorConfigClient      operatorclientv1.ConsoleInterface
	// events
	cachesToSync []cache.InformerSynced
	queue        workqueue.RateLimitingInterface
	recorder     events.Recorder
}

func NewCLIDownloadsSyncController(
	// clients
	operatorClient v1helpers.OperatorClient,
	operatorConfigClient operatorclientv1.OperatorV1Interface,
	cliDownloadsInterface consoleclientv1.ConsoleCLIDownloadInterface,
	routeClient routeclientv1.RoutesGetter,
	// informers
	operatorConfigInformer operatorinformersv1.ConsoleInformer,
	consoleCLIDownloadsInformers consoleinformersv1.ConsoleCLIDownloadInformer,
	routesInformers routesinformersv1.RouteInformer,
	// recorder
	recorder events.Recorder,
) *CLIDownloadsSyncController {

	ctrl := &CLIDownloadsSyncController{
		// clients
		consoleCliDownloadsClient: cliDownloadsInterface,
		routeClient:               routeClient,
		operatorConfigClient:      operatorConfigClient.Consoles(),
		// events
		recorder: recorder,
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ConsoleCliDownloadsSyncer"),
	}

	operatorClient.Informer().AddEventHandler(ctrl.newEventHandler())
	operatorConfigInformer.Informer().AddEventHandler(ctrl.newEventHandler())
	consoleCLIDownloadsInformers.Informer().AddEventHandler(ctrl.newEventHandler())
	routesInformers.Informer().AddEventHandler(ctrl.newEventHandler())

	ctrl.cachesToSync = append(ctrl.cachesToSync,
		operatorClient.Informer().HasSynced,
		operatorConfigInformer.Informer().HasSynced,
		consoleCLIDownloadsInformers.Informer().HasSynced,
		routesInformers.Informer().HasSynced,
	)

	return ctrl
}

func (c *CLIDownloadsSyncController) sync() error {
	operatorConfig, err := c.operatorConfigClient.Get(api.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	updatedOperatorConfig := operatorConfig.DeepCopy()

	switch updatedOperatorConfig.Spec.ManagementState {
	case operatorsv1.Managed:
		klog.V(4).Infoln("console is in a managed state: syncing ConsoleCliDownloads custom resources")
	case operatorsv1.Unmanaged:
		klog.V(4).Infoln("console is in an unmanaged state: skipping ConsoleCliDownloads custom resources sync")
		return nil
	case operatorsv1.Removed:
		klog.V(4).Infoln("console is in a removed state: deleting ConsoleCliDownloads custom resources")
		return c.removeCLIDownloads()
	default:
		return fmt.Errorf("console is in an unknown state: %v", updatedOperatorConfig.Spec.ManagementState)
	}

	consoleRoute, err := c.routeClient.Routes(api.TargetNamespace).Get(api.OpenShiftConsoleDownloadsRouteName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	host := routesub.GetCanonicalHost(consoleRoute)
	ocConsoleCLIDownloads := PlatformBasedOCConsoleCLIDownloads(host, api.OCCLIDownloadsCustomResourceName)
	_, ocCLIDownloadsErrReason, ocCLIDownloadsErr := ApplyCLIDownloads(c.consoleCliDownloadsClient, ocConsoleCLIDownloads)
	status.HandleDegraded(updatedOperatorConfig, "OCDownloadsSync", ocCLIDownloadsErrReason, ocCLIDownloadsErr)
	if ocCLIDownloadsErr != nil {
		return ocCLIDownloadsErr
	}

	_, odoCLIDownloadsErrReason, odoCLIDownloadsErr := ApplyCLIDownloads(c.consoleCliDownloadsClient, ODOConsoleCLIDownloads())
	status.HandleDegraded(updatedOperatorConfig, "ODODownloadsSync", odoCLIDownloadsErrReason, odoCLIDownloadsErr)
	if odoCLIDownloadsErr != nil {
		return odoCLIDownloadsErr
	}

	status.SyncStatus(c.operatorConfigClient, updatedOperatorConfig)

	return nil
}

func (c *CLIDownloadsSyncController) removeCLIDownloads() error {
	defer klog.V(4).Info("finished deleting ConsoleCliDownloads custom resources")
	var errs []error
	errs = append(errs, c.consoleCliDownloadsClient.Delete(api.OCCLIDownloadsCustomResourceName, &metav1.DeleteOptions{}))
	errs = append(errs, c.consoleCliDownloadsClient.Delete(api.ODOCLIDownloadsCustomResourceName, &metav1.DeleteOptions{}))
	return utilerrors.FilterOut(utilerrors.NewAggregate(errs), errors.IsNotFound)
}

func GetPlatformURL(baseURL, platform, archiveType string) string {
	return fmt.Sprintf("%s/%s/%s", baseURL, platform, archiveType)
}

func PlatformBasedOCConsoleCLIDownloads(host, cliDownloadsName string) *v1.ConsoleCLIDownload {
	baseURL := fmt.Sprintf("%s", util.HTTPS(host))
	platforms := []struct {
		label    string
		key      string
		archType string
	}{
		{"Linux for x86_64", "amd64/linux", "oc.tar"},
		{"Mac for x86_64", "amd64/mac", "oc.zip"},
		{"Windows for x86_64", "amd64/windows", "oc.zip"},
		{"Linux for ARM 64 (unsupported)", "arm64/linux", "oc.tar"},
		{"Linux for IBM Power, little endian (unsupported)", "ppc64le/linux", "oc.tar"},
		{"Linux for IBM Z (unsupported)", "s390x/linux", "oc.tar"},
	}

	links := []v1.CLIDownloadLink{}
	for _, platform := range platforms {
		links = append(links, v1.CLIDownloadLink{
			Href: GetPlatformURL(baseURL, platform.key, platform.archType),
			Text: fmt.Sprintf("Download oc for %s", platform.label),
		})
	}

	return &v1.ConsoleCLIDownload{
		ObjectMeta: metav1.ObjectMeta{
			Name: cliDownloadsName,
		},
		Spec: v1.ConsoleCLIDownloadSpec{
			Description: `With the OpenShift command line interface, you can create applications and manage OpenShift projects from a terminal.

The oc binary offers the same capabilities as the kubectl binary, but it is further extended to natively support OpenShift Container Platform features.
`,
			DisplayName: "oc - OpenShift Command Line Interface (CLI)",
			Links:       links,
		},
	}
}

func ODOConsoleCLIDownloads() *v1.ConsoleCLIDownload {
	return &v1.ConsoleCLIDownload{
		ObjectMeta: metav1.ObjectMeta{
			Name: api.ODOCLIDownloadsCustomResourceName,
		},
		Spec: v1.ConsoleCLIDownloadSpec{
			Description: `OpenShift Do (odo) is a fast, iterative, and straightforward CLI tool for developers who write, build, and deploy applications on OpenShift.

odo abstracts away complex Kubernetes and OpenShift concepts, thus allowing developers to focus on what is most important to them: code.
`,
			DisplayName: "odo - Developer-focused CLI for OpenShift",
			Links: []v1.CLIDownloadLink{
				{
					Href: "https://mirror.openshift.com/pub/openshift-v4/clients/odo/latest/",
					Text: "Download odo",
				},
			},
		},
	}
}

// TODO: All the custom `Apply*` functions should be at some point be placed into:
// openshift/library-go/pkg/console/resource/resourceapply/core.go
func ApplyCLIDownloads(consoleClient consoleclientv1.ConsoleCLIDownloadInterface, requiredCLIDownloads *v1.ConsoleCLIDownload) (*v1.ConsoleCLIDownload, string, error) {
	cliDownloadsName := requiredCLIDownloads.ObjectMeta.Name
	existingCLIDownloads, err := consoleClient.Get(cliDownloadsName, metav1.GetOptions{})
	existingCLIDownloadsCopy := existingCLIDownloads.DeepCopy()
	if apierrors.IsNotFound(err) {
		actualCLIDownloads, err := consoleClient.Create(requiredCLIDownloads)
		if err != nil {
			klog.V(4).Infof("error creating %s consoleclidownloads custom resource: %s", cliDownloadsName, err)
			return nil, "FailedCreate", err
		}
		klog.V(4).Infof("%s consoleclidownloads custom resource created", cliDownloadsName)
		return actualCLIDownloads, "", nil
	}
	if err != nil {
		klog.V(4).Infof("error getting %s custom resource: %v", cliDownloadsName, err)
		return nil, "", err
	}
	specSame := equality.Semantic.DeepEqual(existingCLIDownloadsCopy.Spec, requiredCLIDownloads.Spec)
	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existingCLIDownloadsCopy.ObjectMeta, requiredCLIDownloads.ObjectMeta)
	if specSame && !*modified {
		klog.V(4).Infof("%s consoleclidownloads custom resource exists and is in the correct state", cliDownloadsName)
		return existingCLIDownloadsCopy, "", nil
	}

	existingCLIDownloadsCopy.Spec = requiredCLIDownloads.Spec
	actualCLIDownloads, err := consoleClient.Update(existingCLIDownloadsCopy)
	if err != nil {
		klog.V(4).Infof("error updating %s consoleclidownloads custom resource: %v", cliDownloadsName, err)
		return nil, "FailedUpdate", err
	}
	return actualCLIDownloads, "", nil
}

func (c *CLIDownloadsSyncController) Run(workers int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()
	klog.V(4).Infof("Starting %v", controllerName)
	defer klog.V(4).Infof("Shutting down %v", controllerName)
	if !cache.WaitForCacheSync(stopCh, c.cachesToSync...) {
		klog.Infoln("caches did not sync")
		runtime.HandleError(fmt.Errorf("caches did not sync"))
		return
	}
	// only start one worker
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *CLIDownloadsSyncController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *CLIDownloadsSyncController) processNextWorkItem() bool {
	processKey, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(processKey)
	err := c.sync()
	if err == nil {
		c.queue.Forget(processKey)
		return true
	}
	runtime.HandleError(fmt.Errorf("%v failed with : %v", processKey, err))
	c.queue.AddRateLimited(processKey)
	return true
}

func (c *CLIDownloadsSyncController) newEventHandler() cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
		UpdateFunc: func(old, new interface{}) { c.queue.Add(controllerWorkQueueKey) },
		DeleteFunc: func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
	}
}
