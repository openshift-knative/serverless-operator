package knativeservingobsolete

import (
	"context"

	mf "github.com/jcrossley3/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	obsolete "github.com/openshift-knative/serverless-operator/serving/operator/pkg/apis/serving/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	oldapi "github.com/knative/pkg/apis"
	newapi "knative.dev/pkg/apis"
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
	if err := c.Watch(&source.Kind{Type: &obsolete.KnativeServing{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{}); err != nil {
		log.Info("Obsolete KnativeServing CRD not found, and I'm totally cool with that")
		return nil // aborts further setup, we don't need to watch for the new types then either
	}

	// Watch for changes in our "peer".
	if err := c.Watch(&source.Kind{Type: &servingv1alpha1.KnativeServing{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
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
	if err := r.client.Get(context.TODO(), request.NamespacedName, current); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	// Remove finalizers to prevent deadlock
	if len(current.GetFinalizers()) > 0 {
		reqLogger.Info("Removing finalizers for old KnativeServing")
		current.SetFinalizers(nil)
		if err := r.client.Update(context.TODO(), current); err != nil {
			return reconcile.Result{}, err
		}
	}
	// Orphan all the children by removing their OwnerRefs
	if err := r.orphanObsoleteResources(current); err != nil {
		return reconcile.Result{}, err
	}
	// Avoid a certs config conflict in the k-s controller
	if err := r.removeOldCertsConfig(current.Namespace); err != nil {
		return reconcile.Result{}, err
	}

	_, err := r.reconcileNewResource(current)
	if err != nil {
		return reconcile.Result{}, err
	}

<<<<<<< HEAD
	wantStatus := obsolete.KnativeServingStatus{
		Version:    new.Status.Version,
		Conditions: deepCopyConditions(new.Status.Conditions),
	}
	if !equality.Semantic.DeepEqual(current.Status, wantStatus) {
		current.Status = wantStatus
		if err := r.client.Status().Update(context.TODO(), current); err != nil {
			return reconcile.Result{}, err
		}
	}
=======
	// if !equality.Semantic.DeepEqual(current.Status, new.Status) {
	// 	current.Status.Version = new.Status.Version
	// 	current.Status.Conditions = deepCopyConditions(new.Status.Conditions)
	// 	if err := r.client.Status().Update(context.TODO(), current); err != nil {
	// 		return reconcile.Result{}, err
	// 	}
	// 	return reconcile.Result{}, nil
	// }
>>>>>>> 37290bb5... Orphan children without deleting the parent

	return reconcile.Result{}, nil
}

func (r *ReconcileKnativeServingObsolete) reconcileNewResource(old *obsolete.KnativeServing) (*servingv1alpha1.KnativeServing, error) {
	new := &servingv1alpha1.KnativeServing{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: old.Namespace, Name: old.Name}, new)
	if errors.IsNotFound(err) {
		new := &servingv1alpha1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      old.Name,
				Namespace: old.Namespace,
			},
			Spec: servingv1alpha1.KnativeServingSpec{
				Config: old.Spec.Config,
			},
		}
		if err := common.Mutate(new, r.client); err != nil {
			return nil, err
		}
		if err := r.client.Create(context.TODO(), new); err != nil {
			return nil, err
		}
		return new, nil
	} else if err != nil {
		return nil, err
		// } else {
		// 	if !equality.Semantic.DeepEqual(old.Spec.Config, new.Spec.Config) {
		// 		want := new.DeepCopy()
		// 		want.Spec.Config = old.Spec.Config
		// 		if err := common.Mutate(want, r.client); err != nil {
		// 			return nil, err
		// 		}
		// 		if err := r.client.Update(context.TODO(), want); err != nil {
		// 			return nil, err
		// 		}
		// 		return want, nil
		// 	}
	}
	return new, err
}

func (r *ReconcileKnativeServingObsolete) orphanObsoleteResources(ks *obsolete.KnativeServing) error {
	const path = "deploy/resources/knative-serving-0.10.0.yaml"

	manifest, err := mf.NewManifest(path, false, r.client)
	if err != nil {
		return err
	}
	if err := manifest.Transform(mf.InjectNamespace(ks.Namespace)); err != nil {
		return err
	}
	for _, u := range manifest.Resources {
		if u.GetNamespace() != ks.Namespace {
			continue
		}
		resource, err := manifest.Get(&u)
		if err != nil {
			return err
		}
		if resource == nil {
			continue
		}
		for _, owner := range resource.GetOwnerReferences() {
			if owner.UID == ks.UID {
				resource.SetOwnerReferences(nil)
				if err := r.client.Update(context.TODO(), resource); err != nil {
					return err
				}
				log.Info("Orphaned", "name", resource.GetName(), "type", resource.GroupVersionKind())
				break
			}
		}
	}
	return nil

}

// The upstream operator will apply a 3-way strategic merge, leaving
// the old cert config in the controller deployment because we don't
// have the "last-applied" annotation in the 0.10.0 CR from which the
// fields to delete can be determined. Therefore, we'll remove the old
// config ourself.
func (r *ReconcileKnativeServingObsolete) removeOldCertsConfig(ns string) error {
	const name = "controller"
	deployment := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: name}, deployment)
	if errors.IsNotFound(err) {
		// Ignore a not found error, we're not in a migration then.
		return nil
	} else if err != nil {
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

func deepCopyConditions(new []newapi.Condition) []oldapi.Condition {
	old := make([]oldapi.Condition, 0, len(new))
	for _, newCond := range new {
		oldCond := oldapi.Condition{
			Type:               oldapi.ConditionType(string(newCond.Type)),
			Reason:             newCond.Reason,
			Message:            newCond.Message,
			LastTransitionTime: oldapi.VolatileTime{Inner: newCond.LastTransitionTime.Inner},
			Status:             newCond.Status,
			Severity:           oldapi.ConditionSeverity(string(newCond.Severity)),
		}

		old = append(old, oldCond)
	}
	return old
}
