package knativeeventing

import (
	"context"
	"fmt"
	"os"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/consoleutil"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/dashboards/health"
	"github.com/openshift-knative/serverless-operator/pkg/istio/eventingistio"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/dashboards"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
)

const (
	// This needs to remain "knative-eventing-openshift" to be compatible with earlier versions.
	finalizerName = "knative-eventing-openshift"

	requiredNsEnvName = "REQUIRED_EVENTING_NAMESPACE"
)

var log = common.Log.WithName("controller")

// Add creates a new KnativeEventing Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()

	// Create required namespace first.
	if ns, required := os.LookupEnv(requiredNsEnvName); required {
		client.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: ns,
			Labels: map[string]string{
				socommon.ServerlessCommonLabelKey: socommon.ServerlessCommonLabelValue,
			},
		}})
	}

	return &ReconcileKnativeEventing{
		client: client,
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("knativeeventing-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if !consoleutil.IsConsoleInstalled() {
		enqueueRequests := handler.MapFunc(func(obj client.Object) []reconcile.Request {
			if obj.GetName() == consoleutil.ConsoleClusterOperatorName {
				log.Info("Eventing, processing crd request", "name", obj.GetName())
				co := &configv1.ClusterOperator{}
				if err = r.(*ReconcileKnativeEventing).client.Get(context.Background(), client.ObjectKey{Namespace: "", Name: consoleutil.ConsoleClusterOperatorName}, co); err != nil {
					return nil
				}
				if !consoleutil.IsClusterOperatorAvailable(co.Status) {
					return nil
				}
				consoleutil.SetConsoleToInstalledStatus()
				_ = health.InstallHealthDashboard(r.(*ReconcileKnativeEventing).client)
				list := &operatorv1beta1.KnativeEventingList{}
				// At this point we know that console is available and try to find if there is an Eventing instance installed
				// and trigger a reconciliation. If there is no instance do nothing as from now on reconciliation loop will do what is needed
				// when a new instance is created. In case an instance is deleted we do nothing. We read from cache so the call is cheap.
				if err = r.(*ReconcileKnativeEventing).client.List(context.Background(), list); err != nil {
					return nil
				}
				if len(list.Items) > 0 {
					return []reconcile.Request{{
						NamespacedName: types.NamespacedName{Namespace: list.Items[0].Namespace, Name: list.Items[0].Name},
					}}
				}
			}
			return nil
		})
		if err = c.Watch(&source.Kind{Type: &configv1.ClusterOperator{}}, handler.EnqueueRequestsFromMapFunc(enqueueRequests), common.SkipPredicate{}); err != nil {
			return err
		}
	}

	// Watch for changes to primary resource KnativeEventing
	requiredNs := os.Getenv(requiredNsEnvName)
	return c.Watch(&source.Kind{Type: &operatorv1beta1.KnativeEventing{}}, &handler.EnqueueRequestForObject{}, predicate.NewPredicateFuncs(func(obj client.Object) bool {
		if requiredNs == "" {
			return true
		}
		return obj.GetNamespace() == requiredNs
	}))
}

// blank assignment to verify that ReconcileKnativeEventing implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKnativeEventing{}

// ReconcileKnativeEventing reconciles a KnativeEventing object
type ReconcileKnativeEventing struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KnativeEventing
func (r *ReconcileKnativeEventing) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KnativeEventing")

	// Fetch the KnativeEventing instance
	original := &operatorv1beta1.KnativeEventing{}
	err := r.client.Get(context.TODO(), request.NamespacedName, original)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if original.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, r.delete(original)
	}

	instance := original.DeepCopy()
	reconcileErr := r.reconcileKnativeEventing(instance)

	if !equality.Semantic.DeepEqual(original.Status, instance.Status) {
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update status: %w", err)
		}
	}
	monitoring.KnativeEventingUpG = monitoring.KnativeUp.WithLabelValues("eventing_status")
	if instance.Status.IsReady() {
		monitoring.KnativeEventingUpG.Set(1)
	} else {
		monitoring.KnativeEventingUpG.Set(0)
	}
	return reconcile.Result{}, reconcileErr
}

func (r *ReconcileKnativeEventing) reconcileKnativeEventing(instance *operatorv1beta1.KnativeEventing) error {
	stages := []func(*operatorv1beta1.KnativeEventing) error{
		r.maybeScaleIstioController,
		r.ensureFinalizers,
		r.installDashboards,
	}
	for _, stage := range stages {
		if err := stage(instance); err != nil {
			return err
		}
	}
	return nil
}

// set a finalizer to clean up the dashboard when instance is deleted
func (r *ReconcileKnativeEventing) ensureFinalizers(instance *operatorv1beta1.KnativeEventing) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName {
			return nil
		}
	}
	log.Info("Adding finalizer")
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName))
	return r.client.Update(context.TODO(), instance)
}

// installDashboard installs dashboard for OpenShift webconsole
func (r *ReconcileKnativeEventing) installDashboards(instance *operatorv1beta1.KnativeEventing) error {
	if consoleutil.IsConsoleInstalled() {
		log.Info("Installing Eventing Dashboards")
		return dashboards.Apply("eventing", instance, r.client)
	}
	return nil
}

// general clean-up, mostly resources in different namespaces from eventingv1alpha1.KnativeEventing.
func (r *ReconcileKnativeEventing) delete(instance *operatorv1beta1.KnativeEventing) error {
	defer monitoring.KnativeUp.DeleteLabelValues("eventing_status")
	finalizers := sets.NewString(instance.GetFinalizers()...)

	if !finalizers.Has(finalizerName) {
		log.Info("Finalizer has already been removed, nothing to do")
		return nil
	}
	log.Info("Running cleanup logic")
	if consoleutil.IsConsoleInstalled() {
		log.Info("Deleting eventing dashboards")
		if err := dashboards.Delete("eventing", instance, r.client); err != nil {
			return fmt.Errorf("failed to delete resource dashboard configmaps: %w", err)
		}
	}
	// The above might take a while, so we refetch the resource again in case it has changed.
	refetched := &operatorv1beta1.KnativeEventing{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, refetched); err != nil {
		return fmt.Errorf("failed to refetch KnativeEventing: %w", err)
	}

	// Update the refetched finalizer list.
	finalizers = sets.NewString(refetched.GetFinalizers()...)
	finalizers.Delete(finalizerName)
	refetched.SetFinalizers(finalizers.List())

	if err := r.client.Update(context.TODO(), refetched); err != nil {
		return fmt.Errorf("failed to update KnativeEventing with removed finalizer: %w", err)
	}
	return nil
}

func (r *ReconcileKnativeEventing) maybeScaleIstioController(eventing *operatorv1beta1.KnativeEventing) error {
	return eventingistio.MaybeScaleIstioController(r.client, eventing)
}
