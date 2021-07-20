package sources

import (
	"context"
	"os"
	"strconv"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring"
	okomon "github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	v1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	generateSourceServiceMonitorsEnvVar = "GENERATE_SERVICE_MONITORS_FOR_SOURCES_BY_DEFAULT"
	useClusterMonitoringEnvVar          = "USE_CLUSTER_MONITORING_FOR_SOURCES_BY_DEFAULT"

	log = common.Log.WithName("source-deployment-discovery-controller")
)

// Add creates a new Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSourceDeployment{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("source-deployment-discovery-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// common function to enqueue reconcile requests for resources
	enqueueRequests := handler.MapFunc(func(obj client.Object) []reconcile.Request {
		dep := obj.(*v1.Deployment)
		sourceLabel := dep.Spec.Selector.MatchLabels[SourceLabel]
		sourceNameLabel := dep.Spec.Selector.MatchLabels[SourceNameLabel]
		sourceRoleLabel := dep.Spec.Selector.MatchLabels[SourceRoleLabel]

		if (sourceLabel != "" && sourceNameLabel != "") || (sourceLabel != "" && sourceRoleLabel != "") {
			return []reconcile.Request{{
				NamespacedName: types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()},
			}}
		}
		return nil
	})
	err = c.Watch(&source.Kind{Type: &v1.Deployment{}}, handler.EnqueueRequestsFromMapFunc(enqueueRequests))
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileSourceDeployment implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSourceDeployment{}

// ReconcileSourceDeployment reconciles a source deployment object
type ReconcileSourceDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	testShouldUseClusterMonitoring          *bool
	testShouldGenerateSourceServiceMonitors *bool
}

// Reconcile reads that state of the cluster for an eventing source deployment
func (r *ReconcileSourceDeployment) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling the source deployment, setting up a service/service monitor if required")
	dep := &v1.Deployment{}
	err := r.client.Get(context.TODO(), request.NamespacedName, dep)
	inDeletion := false
	if apierrors.IsNotFound(err) {
		// The deployment does not exist anymore, deletions are shown as failing reads
		log.Info("Source in deletion phase")
		inDeletion = true
	} else if err != nil {
		return reconcile.Result{}, err
	}
	eventing := &v1alpha1.KnativeEventing{}
	if err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: "knative-eventing", Name: "knative-eventing"}, eventing); err != nil {
		return reconcile.Result{}, err
	}
	shouldEnableMonitoring := okomon.ShouldEnableMonitoring(eventing.Spec.GetConfig())
	return reconcile.Result{}, r.reconcileSourceMonitoring(dep, shouldEnableMonitoring, inDeletion)
}

func (r *ReconcileSourceDeployment) reconcileSourceMonitoring(dep *v1.Deployment, shouldEnableMonitoring bool, inDeletion bool) error {
	// If monitoring is set to on/off this triggers a global resync to source adapters.
	// Same applies if we change any of the env vars affecting cluster monitoring or service monitor resource generation.
	// The Serverless operator pod is restarted and local informer caches are synched.
	if shouldEnableMonitoring {
		// If in deletion there is nothing to be done, owner refs will remove source service monitors
		if !inDeletion {
			if err := r.setupClusterMonitoringForSources(dep); err != nil {
				return err
			}
			if err := r.generateSourceServiceMonitors(dep); err != nil {
				return err
			}
		}
	} else {
		// Remove any relics if previously monitoring was on.
		if err := r.cleanUpSourceMonitoringResources(dep, inDeletion); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileSourceDeployment) generateSourceServiceMonitors(dep *v1.Deployment) error {
	shouldGenerateSourceMonitors, err := r.shouldGenerateSourceServiceMonitorsByDefault()
	if err != nil {
		return err
	}
	if shouldGenerateSourceMonitors {
		if err := SetupSourceServiceMonitorResources(r.client, dep); err != nil {
			return err
		}
	} else {
		if dep.Namespace != "knative-eventing" {
			if err := RemoveSourceServiceMonitorResources(r.client, dep); err != nil {
				return err
			}
		}
	}
	return nil
}

// Setup cluster monitoring Prometheus monitoring requirements
func (r *ReconcileSourceDeployment) setupClusterMonitoringForSources(dep *v1.Deployment) error {
	shouldEnableClusterMonitoring, err := r.shouldUseClusterMonitoringForSourcesByDefault()
	if err != nil {
		return err
	}
	if shouldEnableClusterMonitoring {
		if err := monitoring.SetupClusterMonitoringRequirements(r.client, dep); err != nil {
			return err
		}
	} else {
		// Make sure we disable cluster monitoring if we have to eg. we move from a state of enabled to disabled and
		// resources are left without cleanup. This brings us to the right state.
		if dep.Namespace != "knative-eventing" {
			if err := monitoring.RemoveClusterMonitoringRequirements(r.client, dep); err != nil {
				return err
			}
		}
	}
	return nil
}

// Remove any Source service monitor resources in any user namespace
func (r *ReconcileSourceDeployment) cleanUpSourceMonitoringResources(dep *v1.Deployment, inDeletion bool) error {
	if dep.Namespace != "knative-eventing" {
		if err := monitoring.RemoveClusterMonitoringRequirements(r.client, dep); err != nil {
			return err
		}
		// No need to do anything if in deletion phase as owner references will do the cleanup
		if !inDeletion {
			if err := RemoveSourceServiceMonitorResources(r.client, dep); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ReconcileSourceDeployment) shouldUseClusterMonitoringForSourcesByDefault() (bool, error) {
	if r.testShouldUseClusterMonitoring != nil {
		return *r.testShouldUseClusterMonitoring, nil
	}
	enable, err := strconv.ParseBool(os.Getenv(useClusterMonitoringEnvVar))
	return enable, err
}

func (r *ReconcileSourceDeployment) shouldGenerateSourceServiceMonitorsByDefault() (bool, error) {
	if r.testShouldGenerateSourceServiceMonitors != nil {
		return *r.testShouldGenerateSourceServiceMonitors, nil
	}
	enable, err := strconv.ParseBool(os.Getenv(generateSourceServiceMonitorsEnvVar))
	return enable, err
}
