package knativeserving

import (
	"context"
	"fmt"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/consoleclidownload"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/kourier"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/apis/istio/v1alpha3"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	routeLabelKey     = "serving.knative.dev/route"
	ingressClassKey   = "networking.knative.dev/ingress.class"
	istioIngressClass = "istio.ingress.networking.knative.dev"
)

var log = common.Log.WithName("controller")

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
	err = c.Watch(&source.Kind{Type: &servingv1alpha1.KnativeServing{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	// Watch for Kourier resources.
	manifest, err := kourier.RawManifest(mgr.GetClient())
	if err != nil {
		return err
	}
	resources := manifest.Resources()

	gvkToKourier := make(map[schema.GroupVersionKind]runtime.Object)
	for i := range resources {
		resource := &resources[i]
		gvkToKourier[resource.GroupVersionKind()] = resource
	}

	for _, t := range gvkToKourier {
		err = c.Watch(&source.Kind{Type: t}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
				annotations := obj.Meta.GetAnnotations()
				ownerNamespace := annotations[kourier.OwnerNamespace]
				ownerName := annotations[kourier.OwnerName]
				if ownerNamespace != "" && ownerName != "" {
					return []reconcile.Request{{
						NamespacedName: types.NamespacedName{Namespace: ownerNamespace, Name: ownerName},
					}}
				}
				return nil
			})})
		if err != nil {
			return err
		}
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
	original := &servingv1alpha1.KnativeServing{}
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
	reconcileErr := r.reconcileKnativeServing(instance)

	if !equality.Semantic.DeepEqual(original.Status, instance.Status) {
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update status: %w", err)
		}
	}
	return reconcile.Result{}, reconcileErr
}

func (r *ReconcileKnativeServing) reconcileKnativeServing(instance *servingv1alpha1.KnativeServing) error {
	stages := []func(*servingv1alpha1.KnativeServing) error{
		r.configure,
		r.ensureFinalizers,
		r.ensureCustomCertsConfigMap,
		r.createConsoleCLIDownload,
		r.installKourier,
		r.updateDeployment,
		r.deleteVirtualService,
	}
	for _, stage := range stages {
		if err := stage(instance); err != nil {
			return err
		}
	}
	return nil
}

// configure default settings for OpenShift
func (r *ReconcileKnativeServing) configure(instance *servingv1alpha1.KnativeServing) error {
	before := instance.DeepCopy()
	if err := common.Mutate(instance, r.client); err != nil {
		return err
	}
	if equality.Semantic.DeepEqual(before, instance) {
		return nil
	}

	// Only apply the update if something changed.
	log.Info("Updating KnativeServing with mutated state for Openshift")
	if err := r.client.Update(context.TODO(), instance); err != nil {
		return fmt.Errorf("failed to update KnativeServing with mutated state: %w", err)
	}
	return nil
}

// deleteVirtualService removes obsoleted VirtualServices.
func (r *ReconcileKnativeServing) deleteVirtualService(instance *servingv1alpha1.KnativeServing) error {
	labelSelector := labels.NewSelector()
	req, err := labels.NewRequirement(routeLabelKey, selection.Exists, nil)
	if err != nil {
		return fmt.Errorf("failed to create requirement for label: %w", err)
	}
	listOpts := &client.ListOptions{LabelSelector: labelSelector.Add(*req)}
	list := &v1alpha3.VirtualServiceList{}
	ctx := context.TODO()
	if err := r.client.List(ctx, listOpts, list); err != nil {
		if meta.IsNoMatchError(err) {
			// VirtualService CRD is not installed.
			return nil
		}
		return err
	}
	for i := range list.Items {
		vs := &list.Items[i]
		if vs.GetAnnotations()[ingressClassKey] == istioIngressClass {
			log.Info(fmt.Sprintf("deleting VirtualService %s/%s", vs.GetNamespace(), vs.GetName()))
			if err := r.client.Delete(ctx, vs); err != nil {
				return fmt.Errorf("failed to delete VirtualService %s/%s: %w", vs.GetNamespace(), vs.GetName(), err)
			}
		}
	}
	return nil
}

// updateDeployment updates Knative controller deployment
func (r *ReconcileKnativeServing) updateDeployment(instance *servingv1alpha1.KnativeServing) error {
	return common.ApplyProxySettings(instance, r.client)
}

// set a finalizer to clean up service mesh when instance is deleted
func (r *ReconcileKnativeServing) ensureFinalizers(instance *servingv1alpha1.KnativeServing) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName() {
			return nil
		}
	}
	log.Info("Adding finalizer")
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName()))
	return r.client.Update(context.TODO(), instance)
}

// create the configmap to be injected with custom certs
func (r *ReconcileKnativeServing) ensureCustomCertsConfigMap(instance *servingv1alpha1.KnativeServing) error {
	certs := instance.Spec.ControllerCustomCerts
	if certs.Type != "ConfigMap" || certs.Name == "" {
		return nil
	}
	cm := &corev1.ConfigMap{}
	ctx := context.TODO()
	if err := r.client.Get(ctx, client.ObjectKey{Name: certs.Name, Namespace: instance.GetNamespace()}, cm); err != nil {
		if errors.IsNotFound(err) {
			cm.Name = certs.Name
			cm.Namespace = instance.GetNamespace()
			cm.Annotations = map[string]string{"service.alpha.openshift.io/inject-cabundle": "true"}
			if err := controllerutil.SetControllerReference(instance, cm, r.scheme); err != nil {
				return err
			}

			log.Info("Creating Custom Certs Config Map")
			if err = r.client.Create(ctx, cm); err != nil {
				return fmt.Errorf("failed to create custom certs config map: %w", err)
			}
			return nil
		}
		return err
	}
	return nil
}

// Install Kourier Ingress Gateway
func (r *ReconcileKnativeServing) installKourier(instance *servingv1alpha1.KnativeServing) error {
	// install Kourier
	return kourier.Apply(instance, r.client, r.scheme)
}

// createConsoleCLIDownload creates CR for kn CLI download link
func (r *ReconcileKnativeServing) createConsoleCLIDownload(instance *servingv1alpha1.KnativeServing) error {
	return consoleclidownload.Create(instance, r.client)
}

// general clean-up, mostly resources in different namespaces from servingv1alpha1.KnativeServing.
func (r *ReconcileKnativeServing) delete(instance *servingv1alpha1.KnativeServing) error {
	finalizer := finalizerName()
	finalizers := sets.NewString(instance.GetFinalizers()...)

	if !finalizers.Has(finalizer) {
		log.Info("Finalizer has already been removed, nothing to do")
		return nil
	}

	log.Info("Running cleanup logic")
	log.Info("Deleting kourier")
	if err := kourier.Delete(instance, r.client, r.scheme); err != nil {
		return fmt.Errorf("failed to delete kourier: %w", err)
	}

	log.Info("Deleting ConsoleCLIDownload")
	if err := consoleclidownload.Delete(instance, r.client); err != nil {
		return fmt.Errorf("failed to delete ConsoleCLIDownload: %w", err)
	}

	// The above might take a while, so we refetch the resource again in case it has changed.
	refetched := &servingv1alpha1.KnativeServing{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, refetched); err != nil {
		return fmt.Errorf("failed to refetch KnativeServing: %w", err)
	}

	// Update the refetched finalizer list.
	finalizers = sets.NewString(refetched.GetFinalizers()...)
	finalizers.Delete(finalizer)
	refetched.SetFinalizers(finalizers.List())

	if err := r.client.Update(context.TODO(), refetched); err != nil {
		return fmt.Errorf("failed to update KnativeServing with removed finalizer: %w", err)
	}
	return nil
}

func finalizerName() string {
	name, err := k8sutil.GetOperatorName()
	if err != nil {
		panic(err)
	}
	return name
}
