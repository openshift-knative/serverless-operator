package knativeserving

import (
	"context"

	mf "github.com/jcrossley3/manifestival"
	"github.com/openshift-knative/knative-serving-openshift/pkg"
	"github.com/openshift-knative/knative-serving-openshift/pkg/controller/knativeserving/servicemesh"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = pkg.Log.WithName("controller")

// Add creates a new KnativeServing Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKnativeServing{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("knativeserving-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KnativeServing
	err = c.Watch(&source.Kind{Type: &servingv1alpha1.KnativeServing{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch servicemesh resources
	err = servicemesh.WatchResources(c)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKnativeServing implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKnativeServing{}

// ReconcileKnativeServing reconciles a KnativeServing object
type ReconcileKnativeServing struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KnativeServing
func (r *ReconcileKnativeServing) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KnativeServing")

	// Fetch the KnativeServing instance
	instance := &servingv1alpha1.KnativeServing{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, r.delete(instance)
	}
	if err := r.ensureFinalizers(instance); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.ensureCustomCertsConfigMap(context.TODO(), instance); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.installNetworkPolicies(context.TODO(), instance); err != nil {
		return reconcile.Result{}, err
	}
	if err := servicemesh.ApplyServiceMesh(instance, r.client); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// create the configmap to be injected with custom certs
func (r *ReconcileKnativeServing) ensureCustomCertsConfigMap(ctx context.Context, instance *servingv1alpha1.KnativeServing) error {
	certs := instance.Spec.ControllerCustomCerts
	if certs.Type != "ConfigMap" || certs.Name == "" {
		return nil
	}
	cm := &corev1.ConfigMap{}
	if err := r.client.Get(ctx, client.ObjectKey{Name: certs.Name, Namespace: instance.GetNamespace()}, cm); err != nil {
		if errors.IsNotFound(err) {
			cm.Name = certs.Name
			cm.Namespace = instance.GetNamespace()
			cm.Annotations = map[string]string{"service.alpha.openshift.io/inject-cabundle": "true"}
			if err := controllerutil.SetControllerReference(instance, cm, r.scheme); err != nil {
				return err
			}
			if err = r.client.Create(ctx, cm); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

// create wide-open networkpolicies for the knative components
func (a *ReconcileKnativeServing) installNetworkPolicies(ctx context.Context, instance *servingv1alpha1.KnativeServing) error {
	namespace := instance.GetNamespace()
	log.Info("Installing Network Policies")
	const path = "deploy/resources/networkpolicies.yaml"

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

func (r *ReconcileKnativeServing) ensureFinalizers(instance *servingv1alpha1.KnativeServing) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName() {
			return nil
		}
	}
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName()))
	return r.client.Update(context.TODO(), instance)
}

func (r *ReconcileKnativeServing) delete(instance *servingv1alpha1.KnativeServing) error {
	if len(instance.GetFinalizers()) == 0 || instance.GetFinalizers()[0] != finalizerName() {
		return nil
	}
	if err := servicemesh.RemoveServiceMesh(instance, r.client); err != nil {
		return err
	}
	// The deletionTimestamp might've changed. Fetch the resource again.
	refetched := &servingv1alpha1.KnativeServing{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, refetched); err != nil {
		return err
	}
	refetched.SetFinalizers(refetched.GetFinalizers()[1:])
	return r.client.Update(context.TODO(), refetched)
}

func finalizerName() string {
	name, err := k8sutil.GetOperatorName()
	if err != nil {
		panic(err)
	}
	return name
}
