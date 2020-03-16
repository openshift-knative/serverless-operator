package resourcesyncdestination

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformersv1 "k8s.io/client-go/informers/core/v1"
	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	operatorsv1 "github.com/openshift/api/operator/v1"
	operatorclientv1 "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	operatorinformersv1 "github.com/openshift/client-go/operator/informers/externalversions/operator/v1"
	"github.com/openshift/console-operator/pkg/api"
	"github.com/openshift/library-go/pkg/operator/events"
)

const (
	controllerWorkQueueKey = "resource-sync-destination-work-queue-key"
	controllerName         = "ConsoleResourceSyncDestinationController"
)

type ResourceSyncDestinationController struct {
	operatorConfigClient operatorclientv1.ConsoleInterface
	configMapClient      coreclientv1.ConfigMapsGetter
	// events
	cachesToSync []cache.InformerSynced
	queue        workqueue.RateLimitingInterface
	recorder     events.Recorder
}

func NewResourceSyncDestinationController(
	// operatorconfig
	operatorConfigClient operatorclientv1.ConsoleInterface,
	operatorConfigInformer operatorinformersv1.ConsoleInformer,
	// configmap
	corev1Client coreclientv1.CoreV1Interface,
	configMapInformer coreinformersv1.ConfigMapInformer,
	// events
	recorder events.Recorder,
) *ResourceSyncDestinationController {
	corev1Client.ConfigMaps(api.OpenShiftConsoleNamespace)

	ctrl := &ResourceSyncDestinationController{
		operatorConfigClient: operatorConfigClient,
		configMapClient:      corev1Client,
		// events
		recorder:     recorder,
		cachesToSync: nil,
		queue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), controllerName),
	}

	configMapInformer.Informer().AddEventHandler(ctrl.newEventHandler())
	operatorConfigInformer.Informer().AddEventHandler(ctrl.newEventHandler())
	ctrl.cachesToSync = append(ctrl.cachesToSync,
		operatorConfigInformer.Informer().HasSynced,
		configMapInformer.Informer().HasSynced,
	)

	return ctrl
}

func (c *ResourceSyncDestinationController) sync() error {
	operatorConfig, err := c.operatorConfigClient.Get(api.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	switch operatorConfig.Spec.ManagementState {
	case operatorsv1.Managed:
		klog.V(4).Infoln("console is in a managed state: syncing default-ingress-cert configmap")
	case operatorsv1.Unmanaged:
		klog.V(4).Infoln("console is in an unmanaged state: skipping default-ingress-cert configmap sync")
		return nil
	case operatorsv1.Removed:
		klog.V(4).Infoln("console is in an removed state: removing synced default-ingress-cert configmap")
		return c.removeDefaultIngressCertConfigMap()
	default:
		return fmt.Errorf("unknown state: %v", operatorConfig.Spec.ManagementState)
	}

	return err
}

func (c *ResourceSyncDestinationController) removeDefaultIngressCertConfigMap() error {
	klog.V(2).Info("deleting default-ingress-cert configmap")
	defer klog.V(2).Info("finished deleting default-ingress-cert configmap")
	return c.configMapClient.ConfigMaps(api.OpenShiftConsoleNamespace).Delete(api.DefaultIngressCertConfigMapName, &metav1.DeleteOptions{})
}

func (c *ResourceSyncDestinationController) Run(workers int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()
	klog.Infof("starting %v", controllerName)
	defer klog.Infof("shutting down %v", controllerName)
	if !cache.WaitForCacheSync(stopCh, c.cachesToSync...) {
		klog.Infoln("caches did not sync")
		runtime.HandleError(fmt.Errorf("caches did not sync"))
		return
	}
	// only start one worker
	go wait.Until(c.runWorker, time.Second, stopCh)
	<-stopCh
}

func (c *ResourceSyncDestinationController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *ResourceSyncDestinationController) processNextWorkItem() bool {
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

func (c *ResourceSyncDestinationController) newEventHandler() cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
		UpdateFunc: func(old, new interface{}) { c.queue.Add(controllerWorkQueueKey) },
		DeleteFunc: func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
	}
}
