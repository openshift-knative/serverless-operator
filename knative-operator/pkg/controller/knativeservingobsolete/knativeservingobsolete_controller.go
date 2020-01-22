package knativeservingobsolete

import (
	"context"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	obsolete "github.com/openshift-knative/serverless-operator/serving/operator/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_knativeservingobsolete")

// Add creates a new KnativeServingObsolete Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKnativeServingObsolete{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("knativeservingobsolete-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KnativeServing
	err = c.Watch(&source.Kind{Type: &obsolete.KnativeServing{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		log.Info("Obsolete KnativeServing CRD not found, and I'm totally cool with that")
	}
	return nil
}

// blank assignment to verify that ReconcileKnativeServingObsolete implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKnativeServingObsolete{}

// ReconcileKnativeServingObsolete reconciles a KnativeServingObsolete object
type ReconcileKnativeServingObsolete struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads creates a new from an old
func (r *ReconcileKnativeServingObsolete) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KnativeServingObsolete")

	// Fetch the KnativeServingObsolete instance
	current := &obsolete.KnativeServing{}
	err := r.client.Get(context.TODO(), request.NamespacedName, current)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	// Fetch the proper instance
	latest := &servingv1alpha1.KnativeServing{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, latest); err == nil {
		// We already have a converted CR, so abort
		return reconcile.Result{}, nil
	}
	// Remove finalizers to prevent deadlock
	if len(current.GetFinalizers()) > 0 {
		reqLogger.Info("Removing finalizers for old KnativeServing")
		current.SetFinalizers(nil)
		if err := r.client.Update(context.TODO(), current); err != nil {
			return reconcile.Result{}, err
		}
	}
	// Create the latest CR from the current (previous) CR
	latest = &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      current.Name,
			Namespace: current.Namespace,
		},
	}
	latest.Spec.Config = current.Spec.Config
	if err := common.Mutate(latest, r.client); err != nil {
		return reconcile.Result{}, err
	}
	// Avoid a certs config conflict in the k-s controller
	if err := r.removeOldCertsConfig(current.Namespace); err != nil {
		return reconcile.Result{}, err
	}
	// Orphan the kids to avoid webhook race condition
	if err := r.client.Delete(context.TODO(), current, client.PropagationPolicy(metav1.DeletePropagationOrphan)); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.client.Create(context.TODO(), latest); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// The upstream operator will apply a 3-way strategic merge, leaving
// the old cert config in the controller deployment because we don't
// have the "last-applied" annotation in the 0.10.0 CR from which the
// fields to delete can be determined. Therefore, we'll remove the old
// config ourself.
func (r *ReconcileKnativeServingObsolete) removeOldCertsConfig(ns string) error {
	const name = "controller"
	deployment := &appsv1.Deployment{}
	if err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: name}, deployment); err != nil {
		return err
	}
	volumes := deployment.Spec.Template.Spec.Volumes
	for i, v := range volumes {
		if v.Name == "service-ca" {
			deployment.Spec.Template.Spec.Volumes = append(volumes[:i], volumes[i+1:]...)
			break
		}
	}
	containers := deployment.Spec.Template.Spec.Containers
	env := containers[0].Env
	for i, v := range env {
		if v.Name == "SSL_CERT_FILE" {
			containers[0].Env = append(env[:i], env[i+1:]...)
			break
		}
	}
	mounts := containers[0].VolumeMounts
	for i, v := range mounts {
		if v.Name == "service-ca" {
			containers[0].VolumeMounts = append(mounts[:i], mounts[i+1:]...)
			break
		}
	}
	if err := r.client.Update(context.TODO(), deployment); err != nil {
		return err
	}
	return nil
}
