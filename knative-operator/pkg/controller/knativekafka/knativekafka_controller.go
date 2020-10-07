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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
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

var log = logf.Log.WithName("controller_knativekafka")

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
	kafkaChannelManifest, err := rawKafkaChannelManifest(mgr.GetClient())
	if err != nil {
		return nil, fmt.Errorf("failed to load KafkaChannel manifest: %w", err)
	}

	kafkaSourceManifest, err := rawKafkaSourceManifest(mgr.GetClient())
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

	stages := []func(*operatorv1alpha1.KnativeKafka) error{
		// TODO r.configure,
		r.ensureFinalizers,
		r.installKnativeKafka,
	}
	for _, stage := range stages {
		if err := stage(instance); err != nil {
			return err
		}
	}
	return nil
}

// set a finalizer to clean up cluster-scoped resources and resources from other namespaces
func (r *ReconcileKnativeKafka) ensureFinalizers(instance *operatorv1alpha1.KnativeKafka) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName {
			return nil
		}
	}
	log.Info("Adding finalizer")
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName))
	return r.client.Update(context.TODO(), instance)
}

// Install Knative Kafka components
func (r *ReconcileKnativeKafka) installKnativeKafka(instance *operatorv1alpha1.KnativeKafka) error {
	if err := r.applyKnativeKafka(instance); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return err
	}
	instance.Status.MarkInstallSucceeded()
	return nil
}

func (r *ReconcileKnativeKafka) applyKnativeKafka(instance *operatorv1alpha1.KnativeKafka) error {
	if instance.Spec.Channel.Enabled {
		if err := r.installKnativeKafkaChannel(instance); err != nil {
			return fmt.Errorf("unable to install Knative KafkaChannel: %w", err)
		}
	} else {
		// TODO: ensure they don't exist
	}

	if instance.Spec.Source.Enabled {
		if err := r.installKnativeKafkaSource(instance); err != nil {
			return fmt.Errorf("unable to install Knative KafkaSource: %w", err)
		}
	} else {
		// TODO: ensure they don't exist
	}

	return nil
}

func (r *ReconcileKnativeKafka) installKnativeKafkaChannel(instance *operatorv1alpha1.KnativeKafka) error {
	manifest, err := r.kafkaChannelManifest(instance)
	if err != nil {
		return err
	}

	log.Info("Installing Knative KafkaChannel")
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply KafkaChannel manifest: %w", err)
	}
	if err := r.checkDeployments(manifest); err != nil {
		return fmt.Errorf("failed to check deployments: %w", err)
	}
	log.Info("Knative KafkaChannel installation is ready")
	return nil
}

// rawKafkaChannelManifest returns KafkaChannel manifest without transformations
func rawKafkaChannelManifest(apiclient client.Client) (mf.Manifest, error) {
	return mfc.NewManifest(kafkaChannelManifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
}

func (r *ReconcileKnativeKafka) kafkaChannelManifest(instance *operatorv1alpha1.KnativeKafka) (*mf.Manifest, error) {
	manifest, err := r.rawKafkaChannelManifest.Transform(
		mf.InjectOwner(instance),
		common.SetAnnotations(map[string]string{
			common.KafkaOwnerName:      instance.Name,
			common.KafkaOwnerNamespace: instance.Namespace,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to transform KafkaChannel manifest: %w", err)
	}

	return &manifest, nil
}

func (r *ReconcileKnativeKafka) installKnativeKafkaSource(instance *operatorv1alpha1.KnativeKafka) error {
	manifest, err := r.kafkaSourceManifest(instance)
	if err != nil {
		return err
	}

	log.Info("Installing Knative KafkaSource")
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply KafkaSource manifest: %w", err)
	}
	if err := r.checkDeployments(manifest); err != nil {
		return fmt.Errorf("failed to check deployments: %w", err)
	}
	log.Info("Knative KafkaSource installation is ready")
	return nil
}

// rawKafkaSourceManifest returns KafkaSource manifest without transformations
func rawKafkaSourceManifest(apiclient client.Client) (mf.Manifest, error) {
	return mfc.NewManifest(kafkaSourceManifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
}

func (r *ReconcileKnativeKafka) kafkaSourceManifest(instance *operatorv1alpha1.KnativeKafka) (*mf.Manifest, error) {
	manifest, err := r.rawKafkaSourceManifest.Transform(
		common.SetAnnotations(map[string]string{
			common.KafkaOwnerName:      instance.Name,
			common.KafkaOwnerNamespace: instance.Namespace,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load KafkaSource manifest: %w", err)
	}

	return &manifest, nil
}

func kafkaChannelManifestPath() string {
	return os.Getenv("KAFKACHANNEL_MANIFEST_PATH")
}

func kafkaSourceManifestPath() string {
	return os.Getenv("KAFKASOURCE_MANIFEST_PATH")
}

// TODO: move to a common place. copied from kourier.go
// Check for deployments
// This function is copied from knativeserving_controller.go in serving-operator
func (r *ReconcileKnativeKafka) checkDeployments(manifest *mf.Manifest) error {
	log.Info("Checking deployments")
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		deployment := &appsv1.Deployment{}
		err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}, deployment)
		if err != nil {
			return err
		}
		for _, c := range deployment.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable && c.Status != corev1.ConditionTrue {
				return fmt.Errorf("Deployment %q/%q not ready", u.GetName(), u.GetNamespace())
			}
		}
	}
	return nil
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
	if instance.Spec.Channel.Enabled {
		if err := r.deleteKnativeKafkaChannel(instance); err != nil {
			return fmt.Errorf("unable to delete Knative KafkaChannel: %w", err)
		}
	}

	if instance.Spec.Source.Enabled {
		if err := r.deleteKnativeKafkaSource(instance); err != nil {
			return fmt.Errorf("unable to delete Knative KafkaSource: %w", err)
		}
	}

	return nil
}

func (r *ReconcileKnativeKafka) deleteKnativeKafkaChannel(instance *operatorv1alpha1.KnativeKafka) error {
	manifest, err := r.kafkaChannelManifest(instance)
	if err != nil {
		return err
	}

	log.Info("Deleting Knative KafkaChannel")

	if err := manifest.Delete(); err != nil {
		return fmt.Errorf("failed to delete Knative KafkaChannel manifest: %w", err)
	}

	return nil
}

func (r *ReconcileKnativeKafka) deleteKnativeKafkaSource(instance *operatorv1alpha1.KnativeKafka) error {
	manifest, err := r.kafkaSourceManifest(instance)
	if err != nil {
		return err
	}

	log.Info("Deleting Knative KafkaSource")
	if err := manifest.Delete(); err != nil {
		return fmt.Errorf("failed to delete KafkaSource manifest: %w", err)
	}
	return nil
}
