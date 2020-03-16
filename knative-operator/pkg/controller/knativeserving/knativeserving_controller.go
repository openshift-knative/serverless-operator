package knativeserving

import (
	"context"
	"fmt"
	"os"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/consoleclidownload"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/kourier"
	consolev1 "github.com/openshift/api/console/v1"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
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

	// This needs to remain "knative-serving-openshift" to be compatible with earlier versions.
	finalizerName = "knative-serving-openshift"

	// serviceCAKey is an annotation key to trigger Openshift to populate service-ca certs to the
	// ConfigMap carrying the annotation.
	// Docs: https://github.com/openshift/service-ca-operator
	serviceCAKey = "service.alpha.openshift.io/inject-cabundle"
	// trustedCAKey is a label key to trigger Openshift to populate trusted CA certs to the
	// ConfigMap carrying the label. This includes CA certs specified in cluster-wide proxy settings.
	// Docs: https://docs.openshift.com/container-platform/4.3/networking/configuring-a-custom-pki.html#certificate-injection-using-operators_configuring-a-custom-pki
	trustedCAKey = "config.openshift.io/inject-trusted-cabundle"

	// certVersionKey is an annotation key used by the Serverless operator to annotate the Knative Serving
	// controller's PodTemplate to make it redeploy on certificate changes.
	certVersionKey = "serving.knative.openshift.io/mounted-cert-version"
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

	// Watch for changes to owned ConfigMaps
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &servingv1alpha1.KnativeServing{},
		IsController: true,
	})
	if err != nil {
		return err
	}

	// common function to enqueue reconcile requests for resources
	enqueueRequests := handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
		annotations := obj.Meta.GetAnnotations()
		ownerNamespace := annotations[common.ServingOwnerNamespace]
		ownerName := annotations[common.ServingOwnerName]
		if ownerNamespace != "" && ownerName != "" {
			return []reconcile.Request{{
				NamespacedName: types.NamespacedName{Namespace: ownerNamespace, Name: ownerName},
			}}
		}
		return nil
	})

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
		err = c.Watch(&source.Kind{Type: t}, &handler.EnqueueRequestsFromMapFunc{ToRequests: enqueueRequests})
		if err != nil {
			return err
		}
	}

	// Watch for kn ConsoleCLIDownload resources
	knManifest, err := consoleclidownload.RawManifest(mgr.GetClient())
	if err != nil {
		return err
	}

	knResources := knManifest.Resources()
	gvkToCCD := make(map[schema.GroupVersionKind]runtime.Object)
	for i := range knResources {
		resource := &knResources[i]
		gvkToCCD[resource.GroupVersionKind()] = resource
	}

	// append ConsoleCLIDownload type as well to Watch for kn CCD CO
	gvkToCCD[consolev1.GroupVersion.WithKind("ConsoleCLIDownload")] = &consolev1.ConsoleCLIDownload{}

	for _, t := range gvkToCCD {
		err = c.Watch(&source.Kind{Type: t}, &handler.EnqueueRequestsFromMapFunc{ToRequests: enqueueRequests})
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
		r.installKnConsoleCLIDownload,
		r.installKourier,
		r.ensureProxySettings,
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

// ensureProxySettings updates the proxy settings on the KnativeServing controller.
func (r *ReconcileKnativeServing) ensureProxySettings(instance *servingv1alpha1.KnativeServing) error {
	proxyEnv := map[string]string{
		"HTTP_PROXY":  os.Getenv("HTTP_PROXY"),
		"HTTPS_PROXY": os.Getenv("HTTPS_PROXY"),
		"NO_PROXY":    os.Getenv("NO_PROXY"),
	}
	return common.ApplyEnvironmentToDeployment(instance.Namespace, "controller", proxyEnv, r.client)
}

// set a finalizer to clean up service mesh when instance is deleted
func (r *ReconcileKnativeServing) ensureFinalizers(instance *servingv1alpha1.KnativeServing) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName {
			return nil
		}
	}
	log.Info("Adding finalizer")
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName))
	return r.client.Update(context.TODO(), instance)
}

// create the configmap to be injected with custom certs
func (r *ReconcileKnativeServing) ensureCustomCertsConfigMap(instance *servingv1alpha1.KnativeServing) error {
	certs := instance.Spec.ControllerCustomCerts

	// If the user doesn't specify anything else, this is set by the webhook/controller defaulter to
	// cause us to automatically pull in the relevant ConfigMaps from the cluster. The user needs
	// to specifically opt-out of this today by specifying an empty Name and ConfigMap explicitly.
	if certs.Type != "ConfigMap" || certs.Name == "" {
		return nil
	}

	serviceCACM, err := r.reconcileConfigMap(instance, certs.Name+"-service-ca", map[string]string{serviceCAKey: "true"}, nil, nil)
	if err != nil {
		return fmt.Errorf("error reconciling serviceCACM: %w", err)
	}
	trustedCACM, err := r.reconcileConfigMap(instance, certs.Name+"-trusted-ca", nil, map[string]string{trustedCAKey: "true"}, nil)
	if err != nil {
		return fmt.Errorf("error reconciling serviceCACM: %w", err)
	}

	combinedContents := make(map[string]string, len(serviceCACM.Data)+len(trustedCACM.Data))
	for key, value := range serviceCACM.Data {
		combinedContents[key] = value
	}
	for key, value := range trustedCACM.Data {
		combinedContents[key] = value
	}

	combinedCM, err := r.reconcileConfigMap(instance, certs.Name, nil, nil, combinedContents)
	if err != nil {
		return fmt.Errorf("error reconciling custom certs CM: %w", err)
	}

	// Check if we need to "kick" the controller deployment.
	controller := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), client.ObjectKey{Namespace: instance.Namespace, Name: "controller"}, controller)
	// If the controller doesn't yet exist, exit early.
	if errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("error fetching controller deployment: %w", err)
	}

	// If the annotation's version is already the latest version, exit early.
	if combinedCM.ResourceVersion == controller.Spec.Template.Annotations[certVersionKey] {
		return nil
	}

	if controller.Spec.Template.Annotations == nil {
		controller.Spec.Template.Annotations = make(map[string]string)
	}

	log.Info("Updating controller cert version",
		"old", controller.Spec.Template.Annotations[certVersionKey], "new", combinedCM.ResourceVersion)

	controller.Spec.Template.Annotations[certVersionKey] = combinedCM.ResourceVersion
	if err := r.client.Update(context.TODO(), controller); err != nil {
		return fmt.Errorf("error updating the controller annotation: %w", err)
	}

	return nil
}

func (r *ReconcileKnativeServing) reconcileConfigMap(instance *servingv1alpha1.KnativeServing, name string,
	annotations, labels map[string]string, data map[string]string) (*corev1.ConfigMap, error) {
	ctx := context.TODO()
	cm := &corev1.ConfigMap{}
	err := r.client.Get(ctx, client.ObjectKey{Name: name, Namespace: instance.GetNamespace()}, cm)
	if errors.IsNotFound(err) {
		cm.Name = name
		cm.Namespace = instance.GetNamespace()
		cm.Annotations = annotations
		cm.Labels = labels
		cm.Data = data
		if err := controllerutil.SetControllerReference(instance, cm, r.scheme); err != nil {
			return nil, fmt.Errorf("failed to set ownerRef on configmap %s: %w", name, err)
		}

		log.Info("Creating config map", "name", name)
		if err = r.client.Create(ctx, cm); err != nil {
			return nil, fmt.Errorf("failed to create config map %s: %w", name, err)
		}
		return cm, nil
	} else if err != nil {
		return nil, err
	} else {
		copy := cm.DeepCopy()
		changed := false
		if !equality.Semantic.DeepEqual(labels, cm.Labels) {
			copy.Labels = labels
			changed = true
		}
		if !equality.Semantic.DeepEqual(annotations, cm.Annotations) {
			copy.Annotations = annotations
			changed = true
		}

		// We only want to interfere with data if we actually desire new data.
		if data != nil && !equality.Semantic.DeepEqual(data, cm.Data) {
			copy.Data = data
			changed = true
		}

		// Only update if we've actually seen a change.
		if changed {
			log.Info("Updating config map", "name", name)
			if err = r.client.Update(ctx, copy); err != nil {
				return nil, fmt.Errorf("failed to update config map %s: %w", name, err)
			}
			return copy, nil
		}
	}
	return cm, nil
}

// Install Kourier Ingress Gateway
func (r *ReconcileKnativeServing) installKourier(instance *servingv1alpha1.KnativeServing) error {
	// install Kourier
	return kourier.Apply(instance, r.client, r.scheme)
}

// installKnConsoleCLIDownload creates CR for kn CLI download link
func (r *ReconcileKnativeServing) installKnConsoleCLIDownload(instance *servingv1alpha1.KnativeServing) error {
	return consoleclidownload.Apply(instance, r.client, r.scheme)
}

// general clean-up, mostly resources in different namespaces from servingv1alpha1.KnativeServing.
func (r *ReconcileKnativeServing) delete(instance *servingv1alpha1.KnativeServing) error {
	finalizers := sets.NewString(instance.GetFinalizers()...)

	if !finalizers.Has(finalizerName) {
		log.Info("Finalizer has already been removed, nothing to do")
		return nil
	}

	log.Info("Running cleanup logic")
	log.Info("Deleting kourier")
	if err := kourier.Delete(instance, r.client, r.scheme); err != nil {
		return fmt.Errorf("failed to delete kourier: %w", err)
	}

	log.Info("Deleting kn ConsoleCLIDownload")
	if err := consoleclidownload.Delete(instance, r.client, r.scheme); err != nil {
		return fmt.Errorf("failed to delete kn ConsoleCLIDownload: %w", err)
	}

	// The above might take a while, so we refetch the resource again in case it has changed.
	refetched := &servingv1alpha1.KnativeServing{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, refetched); err != nil {
		return fmt.Errorf("failed to refetch KnativeServing: %w", err)
	}

	// Update the refetched finalizer list.
	finalizers = sets.NewString(refetched.GetFinalizers()...)
	finalizers.Delete(finalizerName)
	refetched.SetFinalizers(finalizers.List())

	if err := r.client.Update(context.TODO(), refetched); err != nil {
		return fmt.Errorf("failed to update KnativeServing with removed finalizer: %w", err)
	}
	return nil
}
