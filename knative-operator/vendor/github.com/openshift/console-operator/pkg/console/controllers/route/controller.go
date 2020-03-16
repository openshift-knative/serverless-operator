package route

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"time"

	// k8s
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	// openshift
	operatorsv1 "github.com/openshift/api/operator/v1"
	routev1 "github.com/openshift/api/route/v1"
	operatorclientv1 "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	v1 "github.com/openshift/client-go/operator/informers/externalversions/operator/v1"
	routeclientv1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	routesinformersv1 "github.com/openshift/client-go/route/informers/externalversions/route/v1"
	"github.com/openshift/library-go/pkg/operator/events"

	// console-operator
	"github.com/openshift/console-operator/pkg/api"
	customerrors "github.com/openshift/console-operator/pkg/console/errors"
	"github.com/openshift/console-operator/pkg/console/status"
	"github.com/openshift/console-operator/pkg/console/subresource/route"
	routesub "github.com/openshift/console-operator/pkg/console/subresource/route"
)

const (
	controllerName         = "ConsoleRouteSyncController"
	controllerWorkQueueKey = "console-route-sync--work-queue-key"
)

type RouteSyncController struct {
	// clients
	operatorConfigClient operatorclientv1.ConsoleInterface
	routeClient          routeclientv1.RoutesGetter
	configMapClient      coreclientv1.ConfigMapsGetter
	// names
	targetNamespace string
	routeName       string
	// events
	cachesToSync []cache.InformerSynced
	queue        workqueue.RateLimitingInterface
	recorder     events.Recorder
}

func NewRouteSyncController(
	// clients
	operatorConfigClient operatorclientv1.ConsoleInterface,
	routev1Client routeclientv1.RoutesGetter,
	configMapClient coreclientv1.ConfigMapsGetter,
	// informers
	operatorConfigInformer v1.ConsoleInformer,
	routeInformer routesinformersv1.RouteInformer,
	// names
	targetNamespace string,
	routeName string,
	// events
	recorder events.Recorder,
) *RouteSyncController {
	ctrl := &RouteSyncController{
		operatorConfigClient: operatorConfigClient,
		routeClient:          routev1Client,
		configMapClient:      configMapClient,
		targetNamespace:      targetNamespace,
		routeName:            routeName,
		// events
		recorder:     recorder,
		cachesToSync: nil,
		queue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), controllerName),
	}

	operatorConfigInformer.Informer().AddEventHandler(ctrl.newEventHandler())
	routeInformer.Informer().AddEventHandler(ctrl.newEventHandler())

	ctrl.cachesToSync = append(ctrl.cachesToSync,
		operatorConfigInformer.Informer().HasSynced,
		routeInformer.Informer().HasSynced,
	)

	return ctrl
}

func (c *RouteSyncController) sync() error {
	operatorConfig, err := c.operatorConfigClient.Get(api.ConfigResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	switch operatorConfig.Spec.ManagementState {
	case operatorsv1.Managed:
		klog.V(4).Infoln("console is in a managed state: syncing route")
	case operatorsv1.Unmanaged:
		klog.V(4).Infoln("console is in an unmanaged state: skipping route sync")
		return nil
	case operatorsv1.Removed:
		klog.V(4).Infoln("console is in a removed state: deleting route")
		return c.removeRoute()
	default:
		return fmt.Errorf("unknown state: %v", operatorConfig.Spec.ManagementState)
	}

	updatedOperatorConfig := operatorConfig.DeepCopy()
	_, _, errReason, err := c.SyncRoute(updatedOperatorConfig)

	status.HandleProgressingOrDegraded(updatedOperatorConfig, "RouteSync", errReason, err)
	status.SyncStatus(c.operatorConfigClient, updatedOperatorConfig)

	return err
}

func (c *RouteSyncController) removeRoute() error {
	klog.V(2).Info("deleting console route")
	defer klog.V(2).Info("finished deleting console route")
	return c.routeClient.Routes(c.targetNamespace).Delete(route.Stub().Name, &metav1.DeleteOptions{})
}

// apply route
// - be sure to test that we don't trigger an infinite loop by stomping on the
//   default host name set by the server, or any other values. The ApplyRoute()
//   logic will have to be sound.
// - update to ApplyRoute() once the logic is settled
func (c *RouteSyncController) SyncRoute(operatorConfig *operatorsv1.Console) (consoleRoute *routev1.Route, isNew bool, reason string, err error) {
	// ensure we have a route. any error returned is a non-404 error

	rt, rtIsNew, rtErr := routesub.GetOrCreate(c.routeClient, routesub.DefaultRoute(operatorConfig))
	if rtErr != nil {
		return nil, false, "FailedCreate", rtErr
	}
	// Check if the console is reachable
	c.CheckRouteHealth(operatorConfig, rt)

	// we will not proceed until the route is valid. this eliminates complexity with the
	// configmap, secret & oauth client as they can be certain they have a host if we pass this point.
	host := routesub.GetCanonicalHost(rt)
	if len(host) == 0 {
		return nil, false, "FailedHost", customerrors.NewSyncError(fmt.Sprintf("route is not available at canonical host %s", rt.Status.Ingress))
	}

	if validatedRoute, changed := routesub.Validate(rt); changed {
		// if validation changed the route, issue an update
		if _, err := c.routeClient.Routes(api.TargetNamespace).Update(validatedRoute); err != nil {
			// error is unexpected, this is a real error
			return nil, false, "InvalidRouteCorrection", err
		}
		// abort sync, route changed, let it settle & retry
		return nil, true, "InvalidRoute", customerrors.NewSyncError("route is invalid, correcting route state")
	}
	// only return the route if it is valid with a host
	return rt, rtIsNew, "", rtErr
}

func (c *RouteSyncController) Run(workers int, stopCh <-chan struct{}) {
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

func (c *RouteSyncController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *RouteSyncController) processNextWorkItem() bool {
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

func (c *RouteSyncController) newEventHandler() cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
		UpdateFunc: func(old, new interface{}) { c.queue.Add(controllerWorkQueueKey) },
		DeleteFunc: func(obj interface{}) { c.queue.Add(controllerWorkQueueKey) },
	}
}

func (c *RouteSyncController) CheckRouteHealth(opConfig *operatorsv1.Console, rt *routev1.Route) {
	status.HandleDegraded(func() (conf *operatorsv1.Console, prefix string, reason string, err error) {
		prefix = "RouteHealth"

		caPool, err := c.getCA()
		if err != nil {
			return opConfig, prefix, "FailedLoadCA", fmt.Errorf("failed to read CA to check route health: %v", err)
		}
		client := clientWithCA(caPool)

		url := "https://" + rt.Spec.Host + "/health"
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return opConfig, prefix, "FailedRequest", fmt.Errorf("failed to build request to route (%s): %v", url, err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return opConfig, prefix, "FailedGet", fmt.Errorf("failed to GET route (%s): %v", url, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return opConfig, prefix, "StatusError", fmt.Errorf("route not yet available, %s returns '%s'", url, resp.Status)

		}
		return opConfig, prefix, "", nil
	}())

	status.HandleAvailable(opConfig, "Route", "FailedAdmittedIngress", func() error {
		if !routesub.IsAdmitted(rt) {
			return errors.New("console route is not admitted")
		}
		return nil
	}())
}

func (c *RouteSyncController) getCA() (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()

	for _, cmName := range []string{api.TrustedCAConfigMapName, api.DefaultIngressCertConfigMapName} {
		cm, err := c.configMapClient.ConfigMaps(api.OpenShiftConsoleNamespace).Get(cmName, metav1.GetOptions{})
		if err != nil {
			klog.V(4).Infof("failed to GET configmap %s / %s ", api.OpenShiftConsoleNamespace, cmName)
			return nil, err
		}
		if ok := caCertPool.AppendCertsFromPEM([]byte(cm.Data["ca-bundle.crt"])); !ok {
			klog.V(4).Infof("failed to parse %s ca-bundle.crt", cmName)
		}
	}

	return caCertPool, nil
}

func clientWithCA(caPool *x509.CertPool) *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caPool,
			},
		},
	}
}
