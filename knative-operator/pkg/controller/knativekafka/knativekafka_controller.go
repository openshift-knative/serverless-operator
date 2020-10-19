package knativekafka

import (
	"context"
	"fmt"
	"os"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	operatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// DO NOT change to something else in the future!
	// This needs to remain "knative-kafka-openshift" to be compatible with earlier versions in the future versions.
	finalizerName = "knative-kafka-openshift"
)

var (
	log         = logf.Log.WithName("controller_knativekafka")
	role        = mf.Any(mf.ByKind("ClusterRole"), mf.ByKind("Role"))
	rolebinding = mf.Any(mf.ByKind("ClusterRoleBinding"), mf.ByKind("RoleBinding"))
)

type stage func(*mf.Manifest, *operatorv1alpha1.KnativeKafka) error

// Add creates a new KnativeKafka Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (*ReconcileKnativeKafka, error) {
	kafkaChannelManifest, err := mf.ManifestFrom(mf.Path(os.Getenv("KAFKACHANNEL_MANIFEST_PATH")))
	if err != nil {
		return nil, fmt.Errorf("failed to load KafkaChannel manifest: %w", err)
	}

	kafkaSourceManifest, err := mf.ManifestFrom(mf.Path(os.Getenv("KAFKASOURCE_MANIFEST_PATH")))
	if err != nil {
		return nil, fmt.Errorf("failed to load KafkaSource manifest: %w", err)
	}

	reconcileKnativeKafka := ReconcileKnativeKafka{
		client:                  mgr.GetClient(),
		scheme:                  mgr.GetScheme(),
		rawKafkaChannelManifest: kafkaChannelManifest,
		rawKafkaSourceManifest:  kafkaSourceManifest,
	}
	return &reconcileKnativeKafka, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileKnativeKafka) error {
	// Create a new controller
	c, err := controller.New("knativekafka-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KnativeKafka
	err = c.Watch(&source.Kind{Type: &operatorv1alpha1.KnativeKafka{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	gvkToResource := common.BuildGVKToResourceMap(r.rawKafkaChannelManifest, r.rawKafkaSourceManifest)

	for _, t := range gvkToResource {
		err = c.Watch(&source.Kind{Type: t}, common.EnqueueRequestByOwnerAnnotations(common.KafkaOwnerName, common.KafkaOwnerNamespace))
		if err != nil {
			return err
		}
	}

	return nil
}

// blank assignment to verify that ReconcileKnativeKafka implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKnativeKafka{}

// ReconcileKnativeKafka reconciles a KnativeKafka object
type ReconcileKnativeKafka struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client                  client.Client
	scheme                  *runtime.Scheme
	rawKafkaChannelManifest mf.Manifest
	rawKafkaSourceManifest  mf.Manifest
}

// Reconcile reads that state of the cluster for a KnativeKafka object and makes changes based on the state read
// and what is in the KnativeKafka.Spec
func (r *ReconcileKnativeKafka) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KnativeKafka")

	// Fetch the KnativeKafka instance
	original := &operatorv1alpha1.KnativeKafka{}
	err := r.client.Get(context.TODO(), request.NamespacedName, original)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// check for deletion
	if original.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, r.delete(original)
	}

	instance := original.DeepCopy()
	reconcileErr := r.reconcileKnativeKafka(instance)

	if !equality.Semantic.DeepEqual(original.Status, instance.Status) {
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update status: %w", err)
		}
	}

	//TODO: to be enabled
	//if instance.Status.IsReady() {
	//	common.KnativeKafkaUpG.Set(1)
	//} else {
	//	common.KnativeKafkaUpG.Set(0)
	//}
	return reconcile.Result{}, reconcileErr
}

func (r *ReconcileKnativeKafka) reconcileKnativeKafka(instance *operatorv1alpha1.KnativeKafka) error {
	instance.Status.InitializeConditions()

	// install the components that are enabled
	if err := r.executeInstallStages(instance); err != nil {
		return err
	}
	// delete the components that are disabled
	if err := r.executeDeleteStages(instance); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileKnativeKafka) executeInstallStages(instance *operatorv1alpha1.KnativeKafka) error {
	manifest, err := r.buildManifest(instance, manifestBuildEnabledOnly)
	if err != nil {
		return fmt.Errorf("failed to load and build manifest: %w", err)
	}

	stages := []stage{
		r.ensureFinalizers,
		r.transform,
		r.apply,
		r.checkDeployments,
	}

	return executeStages(instance, manifest, stages)
}

func (r *ReconcileKnativeKafka) executeDeleteStages(instance *operatorv1alpha1.KnativeKafka) error {
	manifest, err := r.buildManifest(instance, manifestBuildDisabledOnly)
	if err != nil {
		return fmt.Errorf("failed to load and build manifest: %w", err)
	}

	stages := []stage{
		r.transform,
		r.deleteResources,
	}

	return executeStages(instance, manifest, stages)
}

// set a finalizer to clean up cluster-scoped resources and resources from other namespaces
func (r *ReconcileKnativeKafka) ensureFinalizers(_ *mf.Manifest, instance *operatorv1alpha1.KnativeKafka) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName {
			return nil
		}
	}
	log.Info("Adding finalizer")
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName))
	return r.client.Update(context.TODO(), instance)
}

func (r *ReconcileKnativeKafka) transform(manifest *mf.Manifest, instance *operatorv1alpha1.KnativeKafka) error {
	log.Info("Transforming manifest")
	m, err := manifest.Transform(
		mf.InjectOwner(instance),
		common.SetAnnotations(map[string]string{
			common.KafkaOwnerName:      instance.Name,
			common.KafkaOwnerNamespace: instance.Namespace,
		}),
		setBootstrapServers(instance.Spec.Channel.BootstrapServers),
		ImageTransform(common.BuildImageOverrideMapFromEnviron(os.Environ(), "KAFKA"), log),
	)
	if err != nil {
		return fmt.Errorf("failed to transform manifest: %w", err)
	}
	*manifest = m
	return nil
}

// Install Knative Kafka components
func (r *ReconcileKnativeKafka) apply(manifest *mf.Manifest, instance *operatorv1alpha1.KnativeKafka) error {
	log.Info("Installing manifest")
	// The Operator needs a higher level of permissions if it 'bind's non-existent roles.
	// To avoid this, we strictly order the manifest application as (Cluster)Roles, then
	// (Cluster)RoleBindings, then the rest of the manifest.
	if err := manifest.Filter(role).Apply(); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply (cluster)roles in manifest: %w", err)
	}
	if err := manifest.Filter(rolebinding).Apply(); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply (cluster)rolebindings in manifest: %w", err)
	}
	if err := manifest.Filter(not(mf.Any(role, rolebinding))).Apply(); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply non rbac manifest: %w", err)
	}
	instance.Status.MarkInstallSucceeded()
	return nil
}

func (r *ReconcileKnativeKafka) checkDeployments(manifest *mf.Manifest, instance *operatorv1alpha1.KnativeKafka) error {
	log.Info("Checking deployments")
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		resource, err := manifest.Client.Get(&u)
		if err != nil {
			instance.Status.MarkDeploymentsNotReady()
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		deployment := &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(resource, deployment, nil); err != nil {
			return err
		}
		if !isDeploymentAvailable(deployment) {
			instance.Status.MarkDeploymentsNotReady()
			return nil
		}
	}
	instance.Status.MarkDeploymentsAvailable()
	return nil
}

// Delete Knative Kafka resources
func (r *ReconcileKnativeKafka) deleteResources(manifest *mf.Manifest, instance *operatorv1alpha1.KnativeKafka) error {
	if len(manifest.Resources()) <= 0 {
		return nil
	}
	log.Info("Deleting resources in manifest")
	if err := manifest.Filter(mf.NoCRDs, not(mf.Any(role, rolebinding))).Delete(); err != nil {
		return fmt.Errorf("failed to remove non-crd/non-rbac resources: %w", err)
	}
	// Delete Roles last, as they may be useful for human operators to clean up.
	if err := manifest.Filter(mf.Any(role, rolebinding)).Delete(); err != nil {
		return fmt.Errorf("failed to remove rbac: %w", err)
	}
	return nil
}

func isDeploymentAvailable(d *appsv1.Deployment) bool {
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// general clean-up. required for the resources that cannot be garbage collected with the owner reference mechanism
func (r *ReconcileKnativeKafka) delete(instance *operatorv1alpha1.KnativeKafka) error {
	finalizers := sets.NewString(instance.GetFinalizers()...)

	if !finalizers.Has(finalizerName) {
		log.Info("Finalizer has already been removed, nothing to do")
		return nil
	}

	log.Info("Running cleanup logic")
	log.Info("Deleting KnativeKafka")
	if err := r.deleteKnativeKafka(instance); err != nil {
		return fmt.Errorf("failed to delete KnativeKafka: %w", err)
	}

	// The above might take a while, so we refetch the resource again in case it has changed.
	refetched := &operatorv1alpha1.KnativeKafka{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, refetched); err != nil {
		return fmt.Errorf("failed to refetch KnativeKafka: %w", err)
	}

	// Update the refetched finalizer list.
	finalizers = sets.NewString(refetched.GetFinalizers()...)
	finalizers.Delete(finalizerName)
	refetched.SetFinalizers(finalizers.List())

	if err := r.client.Update(context.TODO(), refetched); err != nil {
		return fmt.Errorf("failed to update KnativeKafka with removed finalizer: %w", err)
	}
	return nil
}

func (r *ReconcileKnativeKafka) deleteKnativeKafka(instance *operatorv1alpha1.KnativeKafka) error {
	manifest, err := r.buildManifest(instance, manifestBuildAll)
	if err != nil {
		return fmt.Errorf("failed to build manifest: %w", err)
	}

	stages := []stage{
		r.transform,
		r.deleteResources,
	}

	return executeStages(instance, manifest, stages)
}

type manifestBuild int

const (
	manifestBuildEnabledOnly manifestBuild = iota
	manifestBuildDisabledOnly
	manifestBuildAll
)

func (r *ReconcileKnativeKafka) buildManifest(instance *operatorv1alpha1.KnativeKafka, build manifestBuild) (*mf.Manifest, error) {
	var resources []unstructured.Unstructured

	if build == manifestBuildAll || (build == manifestBuildEnabledOnly && instance.Spec.Channel.Enabled) || (build == manifestBuildDisabledOnly && !instance.Spec.Channel.Enabled) {
		resources = append(resources, r.rawKafkaChannelManifest.Resources()...)
	}

	if build == manifestBuildAll || (build == manifestBuildEnabledOnly && instance.Spec.Source.Enabled) || (build == manifestBuildDisabledOnly && !instance.Spec.Source.Enabled) {
		resources = append(resources, r.rawKafkaSourceManifest.Resources()...)
	}

	manifest, err := mf.ManifestFrom(
		mf.Slice(resources),
		mf.UseClient(mfc.NewClient(r.client)),
		mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return nil, fmt.Errorf("failed to build Kafka manifest: %w", err)
	}
	return &manifest, nil
}

// setBootstrapServers sets Kafka bootstrapServers value in config-kafka
func setBootstrapServers(bootstrapServers string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "ConfigMap" && u.GetName() == "config-kafka" {
			log.Info("Found ConfigMap config-kafka, updating it with bootstrapServers from spec")
			if err := unstructured.SetNestedField(u.Object, bootstrapServers, "data", "bootstrapServers"); err != nil {
				return err
			}
		}
		return nil
	}
}

func executeStages(instance *operatorv1alpha1.KnativeKafka, manifest *mf.Manifest, stages []stage) error {
	// Execute each stage in sequence until one returns an error
	for _, stage := range stages {
		if err := stage(manifest, instance); err != nil {
			return err
		}
	}
	return nil
}

// TODO: get rid of this when we update to Manifestival version that has this function
var not = func(pred mf.Predicate) mf.Predicate {
	return func(u *unstructured.Unstructured) bool {
		return !pred(u)
	}
}
