package knativekafka

import (
	"context"
	"fmt"
	"os"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"

	operatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

	// NOT IMPLEMENTED YET

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

	// TODO: check for deletion
	// if original.GetDeletionTimestamp() != nil {
	//		return reconcile.Result{}, r.delete(original)
	//	}

	instance := original.DeepCopy()
	reconcileErr := r.reconcileKnativeKafka(instance)

	if !equality.Semantic.DeepEqual(original.Status, instance.Status) {
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update status: %w", err)
		}
	}

	if instance.Status.IsReady() {
		common.KnativeKafkaUpG.Set(1)
	} else {
		common.KnativeKafkaUpG.Set(0)
	}
	return reconcile.Result{}, reconcileErr
}

func (r *ReconcileKnativeKafka) reconcileKnativeKafka(instance *operatorv1alpha1.KnativeKafka) error {
	instance.Status.InitializeConditions()

	stages := []func(*operatorv1alpha1.KnativeKafka) error{
		// TODO r.configure,
		// TODO r.ensureFinalizers,
		r.installKnativeKafka,
	}
	for _, stage := range stages {
		if err := stage(instance); err != nil {
			return err
		}
	}
	return nil
}

// Install Knative Kafka components
func (r *ReconcileKnativeKafka) installKnativeKafka(instance *operatorv1alpha1.KnativeKafka) error {
	if err := applyKnativeKafka(instance, r.client); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return err
	}
	instance.Status.MarkInstallSucceeded()
	return nil
}

func applyKnativeKafka(instance *operatorv1alpha1.KnativeKafka, api client.Client) error {
	if instance.Spec.Channel.Enabled {
		if err := installKnativeKafkaChannel(api); err != nil {
			return fmt.Errorf("unable to install Knative KafkaChannel: %w", err)
		}
	} else {
		// TODO: ensure they don't exist
	}

	if instance.Spec.Source.Enabled {
		if err := installKnativeKafkaSource(api); err != nil {
			return fmt.Errorf("unable to install Knative KafkaSource: %w", err)
		}
	} else {
		// TODO: ensure they don't exist
	}

	return nil
}

func installKnativeKafkaChannel(apiclient client.Client) error {
	manifest, err := mfc.NewManifest(kafkaChannelManifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return fmt.Errorf("failed to load KafkaChannel manifest: %w", err)
	}

	log.Info("Installing Knative KafkaChannel")
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply KafkaChannel manifest: %w", err)
	}
	if err := checkDeployments(&manifest, apiclient); err != nil {
		return fmt.Errorf("failed to check deployments: %w", err)
	}
	log.Info("Knative KafkaChannel installation is ready")
	return nil
}

func installKnativeKafkaSource(apiclient client.Client) error {
	manifest, err := mfc.NewManifest(kafkaSourceManifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
	if err != nil {
		return fmt.Errorf("failed to load KafkaSource manifest: %w", err)
	}

	log.Info("Installing Knative KafkaSource")
	if err := manifest.Apply(); err != nil {
		return fmt.Errorf("failed to apply KafkaSource manifest: %w", err)
	}
	if err := checkDeployments(&manifest, apiclient); err != nil {
		return fmt.Errorf("failed to check deployments: %w", err)
	}
	log.Info("Knative KafkaSource installation is ready")
	return nil
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
func checkDeployments(manifest *mf.Manifest, api client.Client) error {
	log.Info("Checking deployments")
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		deployment := &appsv1.Deployment{}
		err := api.Get(context.TODO(), client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}, deployment)
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
