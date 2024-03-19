package knativeserving

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/consoleclidownload"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/consoleutil"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativeserving/quickstart"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/dashboards"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/dashboards/health"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
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

	// cluster-network-operator enforces the following annotation to be there
	// https://github.com/openshift/cluster-network-operator/pull/2111
	// Otherwise, both operators will revert each other's changes.
	trustedCAOwningAnnotationKey   = "openshift.io/owning-component"
	trustedCAOwningAnnotationValue = "Serverless Operator"

	// certVersionKey is an annotation key used by the Serverless operator to annotate the Knative Serving
	// controller's PodTemplate to make it redeploy on certificate changes.
	certVersionKey = socommon.ServingDownstreamDomain + "/mounted-cert-version"

	requiredNsEnvName = "REQUIRED_SERVING_NAMESPACE"
)

var (
	cliDownloadWatchSet = atomic.Bool{}
	log                 = common.Log.WithName("controller")
)

// Add creates a new KnativeServing Controller and adds it to the Manager. The Manager will set fields on the Controller
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

	return &ReconcileKnativeServing{
		client: client,
		cache:  mgr.GetCache(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("knativeserving-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KnativeServing, only in the expected namespace.
	requiredNs := os.Getenv(requiredNsEnvName)
	err = c.Watch(source.Kind(mgr.GetCache(), &operatorv1beta1.KnativeServing{}), &handler.EnqueueRequestForObject{}, predicate.NewPredicateFuncs(func(obj client.Object) bool {
		if requiredNs == "" {
			return true
		}
		return obj.GetNamespace() == requiredNs
	}))
	if err != nil {
		return err
	}

	// Watch for changes to owned ConfigMaps
	err = c.Watch(source.Kind(mgr.GetCache(), &corev1.ConfigMap{}), handler.EnqueueRequestForOwner(
		mgr.GetScheme(), mgr.GetRESTMapper(), &operatorv1beta1.KnativeServing{}, handler.OnlyControllerOwner()))
	if err != nil {
		return err
	}

	gvkToResource := map[schema.GroupVersionKind]client.Object{
		routev1.GroupVersion.WithKind("Route"): &routev1.Route{},
	}

	// If console is installed add the ccd resource watcher, otherwise remove it avoid manager exiting due to kind not found.
	if consoleutil.IsConsoleInstalled() {
		gvkToResource[consolev1.GroupVersion.WithKind("ConsoleCLIDownload")] = &consolev1.ConsoleCLIDownload{}
	}

	if !consoleutil.IsConsoleInstalled() {
		enqueueRequests := handler.MapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
			if obj.GetName() == consoleutil.ConsoleClusterOperatorName {
				log.Info("Serving, processing clusteroperator request", "name", obj.GetName())
				co := &configv1.ClusterOperator{}
				if err = r.(*ReconcileKnativeServing).client.Get(context.Background(), client.ObjectKey{Namespace: "", Name: consoleutil.ConsoleClusterOperatorName}, co); err != nil {
					return nil
				}
				if !consoleutil.IsClusterOperatorAvailable(co.Status) {
					return nil
				}
				consoleutil.SetConsoleToInstalledStatus()
				_ = health.InstallHealthDashboard(r.(*ReconcileKnativeServing).client)
				list := &operatorv1beta1.KnativeServingList{}
				// At this point we know that console is available and try to find if there is a Serving instance installed
				// and trigger a reconciliation. If there is no instance do nothing as from now on reconciliation loop will do what is needed
				// when a new instance is created. In case an instance is deleted we do nothing. We read from cache so the call is cheap.
				if err = r.(*ReconcileKnativeServing).client.List(context.Background(), list); err != nil {
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
		if err = c.Watch(source.Kind(mgr.GetCache(), &configv1.ClusterOperator{}), handler.EnqueueRequestsFromMapFunc(enqueueRequests), common.SkipPredicate{}); err != nil {
			return err
		}
	}
	for _, t := range gvkToResource {
		err = c.Watch(source.Kind(mgr.GetCache(), t), common.EnqueueRequestByOwnerAnnotations(socommon.ServingOwnerName, socommon.ServingOwnerNamespace))
		if err != nil {
			return err
		}
	}
	r.(*ReconcileKnativeServing).c = &c
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
	cache  cache.Cache
	c      *controller.Controller
}

// Reconcile reads that state of the cluster for a KnativeServing
func (r *ReconcileKnativeServing) Reconcile(_ context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KnativeServing")

	// Fetch the KnativeServing instance
	original := &operatorv1beta1.KnativeServing{}
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

	// If previously not set let's add a watch for ConsoleCLIDownload
	if consoleutil.IsConsoleInstalled() && !cliDownloadWatchSet.Load() {
		if err = (*r.c).Watch(source.Kind(r.cache, &consolev1.ConsoleCLIDownload{}), common.EnqueueRequestByOwnerAnnotations(socommon.ServingOwnerName, socommon.ServingOwnerNamespace)); err != nil {
			return reconcile.Result{}, err
		}
		cliDownloadWatchSet.Store(true)
	}

	instance := original.DeepCopy()
	reconcileErr := r.reconcileKnativeServing(instance)

	if !equality.Semantic.DeepEqual(original.Status, instance.Status) {
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update status: %w", err)
		}
	}

	monitoring.KnativeServingUpG = monitoring.KnativeUp.WithLabelValues("serving_status")
	if instance.Status.IsReady() {
		monitoring.KnativeServingUpG.Set(1)
	} else {
		monitoring.KnativeServingUpG.Set(0)
	}
	return reconcile.Result{}, reconcileErr
}

func (r *ReconcileKnativeServing) reconcileKnativeServing(instance *operatorv1beta1.KnativeServing) error {
	stages := []func(*operatorv1beta1.KnativeServing) error{
		r.ensureFinalizers,
		r.ensureCustomCertsConfigMap,
		r.installDashboard,
		r.installQuickstarts,
		r.installKnConsoleCLIDownload,
	}
	for _, stage := range stages {
		if err := stage(instance); err != nil {
			return err
		}
	}
	return nil
}

// set a finalizer to clean up service mesh when instance is deleted
func (r *ReconcileKnativeServing) ensureFinalizers(instance *operatorv1beta1.KnativeServing) error {
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
func (r *ReconcileKnativeServing) ensureCustomCertsConfigMap(instance *operatorv1beta1.KnativeServing) error {
	certs := instance.Spec.ControllerCustomCerts
	if instance.Spec.ControllerCustomCerts == (base.CustomCerts{}) {
		certs = base.CustomCerts{
			Name: "config-service-ca",
			Type: "ConfigMap",
		}
	}
	if certs.Type != "ConfigMap" || certs.Name == "" {
		return nil
	}

	serviceCACM, err := r.reconcileConfigMap(instance, certs.Name+"-service-ca", map[string]string{serviceCAKey: "true"}, nil, nil)
	if err != nil {
		return fmt.Errorf("error reconciling serviceCACM: %w", err)
	}
	trustedCACM, err := r.reconcileConfigMap(instance, certs.Name+"-trusted-ca",
		map[string]string{trustedCAOwningAnnotationKey: trustedCAOwningAnnotationValue},
		map[string]string{trustedCAKey: "true"}, nil)
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

func (r *ReconcileKnativeServing) reconcileConfigMap(instance *operatorv1beta1.KnativeServing, name string,
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
	}

	cp := cm.DeepCopy()
	changed := false
	if !equality.Semantic.DeepEqual(labels, cm.Labels) {
		cp.Labels = labels
		changed = true
	}
	if !equality.Semantic.DeepEqual(annotations, cm.Annotations) {
		cp.Annotations = annotations
		changed = true
	}

	// We only want to interfere with data if we actually desire new data.
	if data != nil && !equality.Semantic.DeepEqual(data, cm.Data) {
		cp.Data = data
		changed = true
	}

	// Only update if we've actually seen a change.
	if changed {
		log.Info("Updating config map", "name", name)
		if err = r.client.Update(ctx, cp); err != nil {
			return nil, fmt.Errorf("failed to update config map %s: %w", name, err)
		}
		return cp, nil
	}

	return cm, nil
}

func (r *ReconcileKnativeServing) installQuickstarts(_ *operatorv1beta1.KnativeServing) error {
	if consoleutil.IsConsoleInstalled() {
		return quickstart.Apply(r.client)
	}
	return nil
}

// installKnConsoleCLIDownload creates CR for kn CLI download link
func (r *ReconcileKnativeServing) installKnConsoleCLIDownload(instance *operatorv1beta1.KnativeServing) error {
	return consoleclidownload.Apply(instance, r.client)
}

// installDashboard installs dashboard for OpenShift webconsole
func (r *ReconcileKnativeServing) installDashboard(instance *operatorv1beta1.KnativeServing) error {
	if consoleutil.IsConsoleInstalled() {
		log.Info("Installing Serving Dashboards")
		return dashboards.Apply("serving", instance, r.client)
	}
	return nil
}

// general clean-up, mostly resources in different namespaces from servingv1alpha1.KnativeServing.
func (r *ReconcileKnativeServing) delete(instance *operatorv1beta1.KnativeServing) error {
	defer monitoring.KnativeUp.DeleteLabelValues("serving_status")
	finalizers := sets.NewString(instance.GetFinalizers()...)

	if !finalizers.Has(finalizerName) {
		log.Info("Finalizer has already been removed, nothing to do")
		return nil
	}

	log.Info("Running cleanup logic")
	log.Info("Deleting kn ConsoleCLIDownload")
	if err := consoleclidownload.Delete(instance, r.client, r.scheme); err != nil {
		return fmt.Errorf("failed to delete kn ConsoleCLIDownload: %w", err)
	}

	if consoleutil.IsConsoleInstalled() {
		log.Info("Deleting Serving dashboards")
		if err := dashboards.Delete("serving", instance, r.client); err != nil {
			return fmt.Errorf("failed to delete dashboard configmap: %w", err)
		}
	}

	if consoleutil.IsConsoleInstalled() {
		log.Info("Deleting quickstart")
		if err := quickstart.Delete(r.client); err != nil {
			return fmt.Errorf("failed to delete quickstarts: %w", err)
		}
	}

	// The above might take a while, so we refetch the resource again in case it has changed.
	refetched := &operatorv1beta1.KnativeServing{}
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
