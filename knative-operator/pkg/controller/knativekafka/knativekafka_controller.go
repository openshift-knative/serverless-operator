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
	// This needs to remain "knative-kafka-openshift" to be compatible with earlier versions.
	finalizerName = "knative-kafka-openshift"
)

var log = logf.Log.WithName("controller_knativekafka")

// Add creates a new KnativeKafka Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKnativeKafka{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
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

	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &operatorv1alpha1.KnativeKafka{},
		IsController: true,
	})
	if err != nil {
		return err
	}

	// Load Knative KafkaChannel resources to watch them
	kafkaChannelManifest, err := rawKafkaChannelManifest(mgr.GetClient())
	if err != nil {
		return err
	}

	// Load Knative KafkaSource resources to watch them
	kafkaSourceManifest, err := rawKafkaSourceManifest(mgr.GetClient())
	if err != nil {
		return err
	}

	gvkToResource := common.BuildGVKToResourceMap(kafkaChannelManifest, kafkaSourceManifest)

	// common function to enqueue reconcile requests for resources
	enqueueRequests := common.EnqueueRequestByOwnerAnnotations(common.KafkaOwnerName, common.KafkaOwnerNamespace)
	for _, t := range gvkToResource {
		err = c.Watch(&source.Kind{Type: t}, &handler.EnqueueRequestsFromMapFunc{ToRequests: enqueueRequests})
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
	client client.Client
	scheme *runtime.Scheme
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
	manifest, err := buildManifest(instance, r.client, ManifestBuildEnabledOnly)
	if err != nil {
		return fmt.Errorf("failed to load and build manifest: %w", err)
	}

	stages := []func(*mf.Manifest, *operatorv1alpha1.KnativeKafka) error{
		// TODO r.configure,
		r.ensureFinalizers,
		r.transform,
		r.apply,
		r.checkDeployments,
	}

	// Execute each stage in sequence until one returns an error
	for _, stage := range stages {
		if err := stage(manifest, instance); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileKnativeKafka) executeDeleteStages(instance *operatorv1alpha1.KnativeKafka) error {
	manifest, err := buildManifest(instance, r.client, ManifestBuildDisabledOnly)
	if err != nil {
		return fmt.Errorf("failed to load and build manifest: %w", err)
	}

	stages := []func(*mf.Manifest, *operatorv1alpha1.KnativeKafka) error{
		r.transform,
		r.deleteResources,
	}

	// Execute each stage in sequence until one returns an error
	for _, stage := range stages {
		if err := stage(manifest, instance); err != nil {
			return err
		}
	}
	return nil
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
	transformers := []mf.Transformer{
		InjectOwner(instance),
		common.SetOwnerAnnotations(instance.ObjectMeta, common.KafkaOwnerName, common.KafkaOwnerNamespace),
	}

	log.Info("Transforming manifest")
	m, err := manifest.Transform(transformers...)
	if err != nil {
		return fmt.Errorf("failed to transform manifest: %w", err)
	}
	*manifest = m
	return nil
}

// Install Knative Kafka components
func (r *ReconcileKnativeKafka) apply(manifest *mf.Manifest, instance *operatorv1alpha1.KnativeKafka) error {
	log.Info("Installing manifest")
	if err := manifest.Apply(); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply manifest: %w", err)
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
	log.Info("Deleting resources in manifest")
	if err := manifest.Delete(); err != nil {
		// TODO: any conditions?
		return fmt.Errorf("failed to apply manifest: %w", err)
	}
	// TODO: any conditions?
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

// rawKafkaChannelManifest returns KafkaChannel manifest without transformations
func rawKafkaChannelManifest(apiclient client.Client) (mf.Manifest, error) {
	return mfc.NewManifest(kafkaChannelManifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
}

// rawKafkaSourceManifest returns KafkaSource manifest without transformations
func rawKafkaSourceManifest(apiclient client.Client) (mf.Manifest, error) {
	return mfc.NewManifest(kafkaSourceManifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
}

func kafkaChannelManifestPath() string {
	return os.Getenv("KAFKACHANNEL_MANIFEST_PATH")
}

func kafkaSourceManifestPath() string {
	return os.Getenv("KAFKASOURCE_MANIFEST_PATH")
}

// general clean-up. required for the resources that cannot be garbage collected with the owner reference mechanism
func (r *ReconcileKnativeKafka) delete(instance *operatorv1alpha1.KnativeKafka) error {
	finalizers := sets.NewString(instance.GetFinalizers()...)

	if !finalizers.Has(finalizerName) {
		log.Info("Finalizer has already been removed, nothing to do")
		return nil
	}

	log.Info("Running cleanup logic")
	log.Info("Deleting Knative Kafka")
	if err := r.deleteKnativeKafka(instance); err != nil {
		return fmt.Errorf("failed to delete Knative Kafka: %w", err)
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
	manifest, err := buildManifest(instance, r.client, ManifestBuildAll)
	if err != nil {
		return fmt.Errorf("failed to load and build manifest: %w", err)
	}

	stages := []func(*mf.Manifest, *operatorv1alpha1.KnativeKafka) error{
		r.transform,
		r.deleteResources,
	}

	// Execute each stage in sequence until one returns an error
	for _, stage := range stages {
		if err := stage(manifest, instance); err != nil {
			return err
		}
	}
	return nil
}

type manifestBuild int

const (
	ManifestBuildEnabledOnly manifestBuild = iota
	ManifestBuildDisabledOnly
	ManifestBuildAll
)

func buildManifest(instance *operatorv1alpha1.KnativeKafka, apiClient client.Client, build manifestBuild) (*mf.Manifest, error) {
	combinedManifest := &mf.Manifest{}

	if build == ManifestBuildAll || (instance.Spec.Channel.Enabled && build == ManifestBuildEnabledOnly) || (!instance.Spec.Channel.Enabled && build == ManifestBuildDisabledOnly) {
		manifest, err := rawKafkaSourceManifest(apiClient)
		if err != nil {
			return nil, fmt.Errorf("failed to load KafkaChannel manifest: %w", err)
		}
		combinedManifest, err = mergeManifests(manifest.Client, combinedManifest, &manifest)
		if err != nil {
			return nil, fmt.Errorf("failed to merge KafkaChannel manifest: %w", err)
		}
	}

	if build == ManifestBuildAll || (instance.Spec.Source.Enabled && build == ManifestBuildEnabledOnly) || (!instance.Spec.Source.Enabled && build == ManifestBuildDisabledOnly) {
		manifest, err := rawKafkaSourceManifest(apiClient)
		if err != nil {
			return nil, fmt.Errorf("failed to load KafkaSource manifest: %w", err)
		}
		combinedManifest, err = mergeManifests(manifest.Client, combinedManifest, &manifest)
		if err != nil {
			return nil, fmt.Errorf("failed to merge KafkaSource manifest: %w", err)
		}
	}
	return combinedManifest, nil
}

// Merges the given manifests into a new single manifest
func mergeManifests(client mf.Client, m1, m2 *mf.Manifest) (*mf.Manifest, error) {
	result, err := mf.ManifestFrom(mf.Slice(append(m1.Resources(), m2.Resources()...)))
	if err != nil {
		return nil, fmt.Errorf("failed to merge manifests: %w", err)
	}
	result.Client = client
	return &result, nil
}

// InjectOwner creates a Tranformer which adds an OwnerReference pointing to
// `owner` to namespace-scoped objects.
//
// The difference from Manifestival's Inject owner is, it only does it for
// resources that are in the same namespace as the owner.
// For the resources that are in the same namespace, it fallbacks to
// Manifestival's InjectOwner
func InjectOwner(owner mf.Owner) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetNamespace() == owner.GetNamespace() {
			return mf.InjectOwner(owner)(u)
		} else {
			return nil
		}
	}
}
