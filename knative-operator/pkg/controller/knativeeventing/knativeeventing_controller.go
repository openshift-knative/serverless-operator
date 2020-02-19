package knativeeventing

import (
	"context"
	mf "github.com/jcrossley3/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	eventingv1alpha1 "knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = common.Log.WithName("controller")

// Add creates a new KnativeEventing Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKnativeEventing{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("knativeeventing-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KnativeEventing
	err = c.Watch(&source.Kind{Type: &eventingv1alpha1.KnativeEventing{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	return nil
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
func (r *ReconcileKnativeEventing) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KnativeEventing")

	// Fetch the KnativeEventing instance
	instance := &eventingv1alpha1.KnativeEventing{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// TODO: I assumed we don't need this
	//if instance.GetDeletionTimestamp() != nil {
	//	return reconcile.Result{}, r.delete(instance)
	//}

	stages := []func(*eventingv1alpha1.KnativeEventing) error{
		r.configure,
		r.installNetworkPolicies,
	}
	for _, stage := range stages {
		if err := stage(instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// configure default settings for OpenShift
func (r *ReconcileKnativeEventing) configure(instance *eventingv1alpha1.KnativeEventing) error {
	if _, ok := instance.GetAnnotations()[common.MutationTimestampKey]; ok {
		return nil
	}
	log.Info("Configuring KnativeEventing for OpenShift")
	if err := common.MutateEventing(instance, r.client); err != nil {
		return err
	}
	return r.client.Update(context.TODO(), instance)
}

// create wide-open networkpolicies for the knative components
func (a *ReconcileKnativeEventing) installNetworkPolicies(instance *eventingv1alpha1.KnativeEventing) error {
	namespace := instance.GetNamespace()
	log.Info("Installing Network Policies")
	const path = "deploy/resources/eventing_networkpolicies.yaml"

	manifest, err := mf.NewManifest(path, false, a.client)
	if err != nil {
		return err
	}
	transforms := []mf.Transformer{mf.InjectOwner(instance)}
	if len(namespace) > 0 {
		transforms = append(transforms, mf.InjectNamespace(namespace))
	}
	if err := manifest.Transform(transforms...); err != nil {
		return err
	}
	if err := manifest.ApplyAll(); err != nil {
		return err
	}
	return nil
}
