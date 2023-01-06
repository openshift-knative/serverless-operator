package knativekafka

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	socommon "github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"

	"sigs.k8s.io/controller-runtime/pkg/source"

	operatorcommon "knative.dev/operator/pkg/reconciler/common"
	"knative.dev/pkg/logging"

	mfc "github.com/manifestival/controller-runtime-client"

	"knative.dev/operator/pkg/apis/operator/base"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	serverlessoperatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring"
)

const (
	// DO NOT change to something else in the future!
	// This needs to remain "knative-kafka-openshift" to be compatible with earlier versions in the future versions.
	finalizerName = "knative-kafka-openshift"
)

var (
	log               = logf.Log.WithName("controller_knativekafka")
	role              = mf.Any(mf.ByKind("ClusterRole"), mf.ByKind("Role"))
	rolebinding       = mf.Any(mf.ByKind("ClusterRoleBinding"), mf.ByKind("RoleBinding"))
	roleOrRoleBinding = mf.Any(role, rolebinding)
	KafkaHAComponents = []string{"kafka-controller", "kafka-webhook-eventing"}

	dependentConfigMaps = sets.NewString("config-tracing", "kafka-config-logging")
)

type stage func(*mf.Manifest, *serverlessoperatorv1alpha1.KnativeKafka) error

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

	kafkaControllerManifest, err := mf.ManifestFrom(mf.Path(os.Getenv("KAFKACONTROLLER_MANIFEST_PATH")))
	if err != nil {
		return nil, fmt.Errorf("failed to load Kafka Control-Plane manifest: %w", err)
	}

	kafkaBrokerManifest, err := mf.ManifestFrom(mf.Path(os.Getenv("KAFKABROKER_MANIFEST_PATH")))
	if err != nil {
		return nil, fmt.Errorf("failed to load KafkaBroker manifest: %w", err)
	}

	kafkaSinkManifest, err := mf.ManifestFrom(mf.Path(os.Getenv("KAFKASINK_MANIFEST_PATH")))
	if err != nil {
		return nil, fmt.Errorf("failed to load KafkaBroker manifest: %w", err)
	}

	reconcileKnativeKafka := ReconcileKnativeKafka{
		client:                     mgr.GetClient(),
		scheme:                     mgr.GetScheme(),
		rawKafkaChannelManifest:    kafkaChannelManifest,
		rawKafkaSourceManifest:     kafkaSourceManifest,
		rawKafkaControllerManifest: kafkaControllerManifest,
		rawKafkaBrokerManifest:     kafkaBrokerManifest,
		rawKafkaSinkManifest:       kafkaSinkManifest,
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
	err = c.Watch(&source.Kind{Type: &serverlessoperatorv1alpha1.KnativeKafka{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	gvkToResource := common.BuildGVKToResourceMap(
		r.rawKafkaControllerManifest,
		r.rawKafkaBrokerManifest,
		r.rawKafkaChannelManifest,
		r.rawKafkaSourceManifest,
		r.rawKafkaSinkManifest,
	)

	for _, t := range gvkToResource {
		err = c.Watch(&source.Kind{Type: t}, common.EnqueueRequestByOwnerAnnotations(common.KafkaOwnerName, common.KafkaOwnerNamespace))
		if err != nil {
			return err
		}
	}

	// Add eventing core resources to watch
	gvkToEventingResource := map[schema.GroupVersionKind]client.Object{
		corev1.SchemeGroupVersion.WithKind("ConfigMap"): &corev1.ConfigMap{},
	}

	for _, t := range gvkToEventingResource {
		err = c.Watch(&source.Kind{Type: t}, filteredGlobalResync(context.Background(), mgr.GetLogger(), r, dependentConfigMaps))
		if err != nil {
			return err
		}
	}

	return nil
}

func filteredGlobalResync(ctx context.Context, logger logr.Logger, r *ReconcileKnativeKafka, names sets.String) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
		ns := "knative-eventing"
		if object.GetNamespace() != ns || !names.Has(object.GetName()) {
			return nil
		}

		logger.Info("Global resync",
			"object.namespace", object.GetNamespace(),
			"object.name", object.GetName(),
		)

		kks := &serverlessoperatorv1alpha1.KnativeKafkaList{}
		if err := r.client.List(ctx, kks); err != nil {
			log.Error(err, "failed to list KnativeKafka objects")
		}
		requests := make([]reconcile.Request, 0, len(kks.Items))
		for _, kk := range kks.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: kk.GetNamespace(), Name: kk.GetName()},
			})
		}
		return requests
	})
}

// blank assignment to verify that ReconcileKnativeKafka implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKnativeKafka{}

// ReconcileKnativeKafka reconciles a KnativeKafka object
type ReconcileKnativeKafka struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client                     client.Client
	scheme                     *runtime.Scheme
	rawKafkaChannelManifest    mf.Manifest
	rawKafkaSourceManifest     mf.Manifest
	rawKafkaControllerManifest mf.Manifest
	rawKafkaBrokerManifest     mf.Manifest
	rawKafkaSinkManifest       mf.Manifest
}

// Reconcile reads that state of the cluster for a KnativeKafka object and makes changes based on the state read
// and what is in the KnativeKafka.Spec
func (r *ReconcileKnativeKafka) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KnativeKafka")

	// Fetch the KnativeKafka instance
	original := &serverlessoperatorv1alpha1.KnativeKafka{}
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

	monitoring.KnativeKafkaUpG = monitoring.KnativeUp.WithLabelValues("kafka_status")
	if instance.Status.IsReady() {
		monitoring.KnativeKafkaUpG.Set(1)
	} else {
		monitoring.KnativeKafkaUpG.Set(0)
	}
	return reconcile.Result{}, reconcileErr
}

func (r *ReconcileKnativeKafka) reconcileKnativeKafka(instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	instance.Status.InitializeConditions()

	// install the components that are enabled
	if err := r.executeInstallStages(instance); err != nil {
		return err
	}
	// delete the components that are disabled
	return r.executeDeleteStages(instance)
}

func (r *ReconcileKnativeKafka) executeInstallStages(instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	manifest, err := r.buildManifest(instance, manifestBuildEnabledOnly)
	if err != nil {
		return fmt.Errorf("failed to load and build manifest: %w", err)
	}

	stages := []stage{
		r.configure,
		r.ensureFinalizers,
		r.transform,
		removeCreationTimestamp,
		r.apply,
		r.checkDeployments,
		r.checkStatefulSets,
	}

	return executeStages(instance, manifest, stages)
}

func (r *ReconcileKnativeKafka) executeDeleteStages(instance *serverlessoperatorv1alpha1.KnativeKafka) error {
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

// set defaults for Openshift
func (r *ReconcileKnativeKafka) configure(manifest *mf.Manifest, instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	if instance.Spec.HighAvailability == nil {
		instance.Spec.HighAvailability = &base.HighAvailability{
			Replicas: ptr.Int32(1),
		}
	}

	// using INFO logging is reasonable default
	if instance.Spec.Logging == nil {
		instance.Spec.Logging = &serverlessoperatorv1alpha1.Logging{
			Level: serverlessoperatorv1alpha1.DefaultLogLevel,
		}
	}

	return nil
}

// set a finalizer to clean up cluster-scoped resources and resources from other namespaces
func (r *ReconcileKnativeKafka) ensureFinalizers(manifest *mf.Manifest, instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName {
			return nil
		}
	}
	log.Info("Adding finalizer")
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName))
	return r.client.Update(context.TODO(), instance)
}

func (r *ReconcileKnativeKafka) transform(manifest *mf.Manifest, instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	log.Info("Transforming manifest")
	// If in deletion we don't apply any monitoring transformer to kafka components and transformer will be nil and skipped.
	var rbacProxyTranform mf.Transformer
	if instance.GetDeletionTimestamp() == nil {
		var err error
		if rbacProxyTranform, err = monitoring.GetRBACProxyInjectTransformer(r.client); err != nil {
			return err
		}
	}
	tfs := []mf.Transformer{}
	tfs = append(append(tfs,
		mf.InjectOwner(instance),
		common.SetAnnotations(map[string]string{
			common.KafkaOwnerName:      instance.Name,
			common.KafkaOwnerNamespace: instance.Namespace,
		}),
		setKafkaDeployments(*instance.Spec.HighAvailability.Replicas),
		operatorcommon.ConfigMapTransform(instance.Spec.Config, logging.FromContext(context.TODO())),
		configureEventingKafka(instance.Spec),
		ImageTransform(common.BuildImageOverrideMapFromEnviron(os.Environ(), "KAFKA_IMAGE_")),
		socommon.VersionedJobNameTransform(),
		socommon.InjectCommonEnvironment(),
		operatorcommon.OverridesTransform(instance.Spec.Workloads, logging.FromContext(context.TODO())),
		socommon.ConfigMapVolumeChecksumTransform(context.Background(), r.client, sets.NewString("config-tracing", "kafka-config-logging")),
		rbacProxyTranform), socommon.DeprecatedAPIsTranformersFromConfig()...)
	m, err := manifest.Transform(tfs...)
	if err != nil {
		return fmt.Errorf("failed to transform manifest: %w", err)
	}
	*manifest = m
	return nil
}

// Install Knative Kafka components
func (r *ReconcileKnativeKafka) apply(manifest *mf.Manifest, instance *serverlessoperatorv1alpha1.KnativeKafka) error {
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
	if err := manifest.Filter(mf.Not(roleOrRoleBinding)).Apply(); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply non rbac manifest: %w", err)
	}
	instance.Status.MarkInstallSucceeded()
	instance.Status.Version = os.Getenv("KNATIVE_EVENTING_KAFKA_BROKER_VERSION")
	return nil
}

func (r *ReconcileKnativeKafka) checkDeployments(manifest *mf.Manifest, instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	log.Info("Checking deployments")
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		u := u // To avoid memory aliasing
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

func (r *ReconcileKnativeKafka) checkStatefulSets(manifest *mf.Manifest, instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	log.Info("Checking statefulsets")
	for _, u := range manifest.Filter(mf.ByKind("StatefulSet")).Resources() {
		u := u // To avoid memory aliasing
		resource, err := manifest.Client.Get(&u)
		if err != nil {
			instance.Status.MarkStatefulSetNotReady()
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		ss := &appsv1.StatefulSet{}
		if err := scheme.Scheme.Convert(resource, ss, nil); err != nil {
			return err
		}
		if !isStatefulSetAvailable(ss) {
			instance.Status.MarkStatefulSetNotReady()
			return nil
		}
	}
	instance.Status.MarkStatefulSetsAvailable()
	return nil
}

// Delete Knative Kafka resources
func (r *ReconcileKnativeKafka) deleteResources(manifest *mf.Manifest, instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	if len(manifest.Resources()) <= 0 {
		return nil
	}
	log.Info("Deleting resources in manifest")
	if err := manifest.Filter(mf.NoCRDs, mf.Not(roleOrRoleBinding)).Delete(); err != nil {
		return fmt.Errorf("failed to remove non-crd/non-rbac resources: %w", err)
	}
	// Delete Roles last, as they may be useful for human operators to clean up.
	if err := manifest.Filter(roleOrRoleBinding).Delete(); err != nil {
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

func isStatefulSetAvailable(ss *appsv1.StatefulSet) bool {
	if ss.Spec.Replicas == nil {
		return ss.Status.ReadyReplicas == 1
	}
	// On upgrades and during normal operation the number of replicas might vary, however, we
	// shouldn't transition to "not ready" just because there are some non-ready replicas because that
	// could  trigger a warning for anyone using KnativeKafka that it's not a signal of a
	// non-functional installation.
	//
	// Note: we do this only in the case where the number of expected replicas is >= 1, so that we
	// make sure that at least one replica is ready and functional.
	if *ss.Spec.Replicas >= 1 {
		return ss.Status.ReadyReplicas >= 1
	}
	return *ss.Spec.Replicas == ss.Status.ReadyReplicas
}

// general clean-up. required for the resources that cannot be garbage collected with the owner reference mechanism
func (r *ReconcileKnativeKafka) delete(instance *serverlessoperatorv1alpha1.KnativeKafka) error {
	defer monitoring.KnativeUp.DeleteLabelValues("kafka_status")
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
	refetched := &serverlessoperatorv1alpha1.KnativeKafka{}
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

func (r *ReconcileKnativeKafka) deleteKnativeKafka(instance *serverlessoperatorv1alpha1.KnativeKafka) error {
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
	brokerController                       = "BROKER"
	sinkController                         = "SINK"
	sourceController                       = "SOURCE"
	channelController                      = "CHANNEL"
	manifestBuildEnabledOnly manifestBuild = iota
	manifestBuildDisabledOnly
	manifestBuildAll
)

func (r *ReconcileKnativeKafka) buildManifest(instance *serverlessoperatorv1alpha1.KnativeKafka, build manifestBuild) (*mf.Manifest, error) {
	var resources []unstructured.Unstructured

	if build == manifestBuildAll || (build == manifestBuildEnabledOnly && instance.Spec.Channel.Enabled) || (build == manifestBuildDisabledOnly && !instance.Spec.Channel.Enabled) {
		rbacProxy, err := monitoring.AddRBACProxyToManifest(instance, monitoring.KafkaChannelReceiver, monitoring.KafkaChannelDispatcher)
		if err != nil {
			return nil, err
		}
		resources = append(resources, rbacProxy.Resources()...)
		resources = append(resources, r.rawKafkaChannelManifest.Resources()...)
	}

	// Kafka Control Plane
	if build == manifestBuildAll || (build == manifestBuildEnabledOnly && enableControlPlaneManifest(instance.Spec)) || (build == manifestBuildDisabledOnly && !enableControlPlaneManifest(instance.Spec)) {
		rbacProxy, err := monitoring.AddRBACProxyToManifest(instance, monitoring.KafkaController, monitoring.KafkaWebhook)
		if err != nil {
			return nil, err
		}
		resources = append(resources, rbacProxy.Resources()...)
		resources = append(resources, r.rawKafkaControllerManifest.Resources()...)
	}

	// Kafka Source Data Plane
	if build == manifestBuildAll || (build == manifestBuildEnabledOnly && instance.Spec.Source.Enabled) || (build == manifestBuildDisabledOnly && !instance.Spec.Source.Enabled) {
		sourceRBACProxy, err := monitoring.AddRBACProxyToManifest(instance, monitoring.KafkaSourceDispatcher)
		if err != nil {
			return nil, err
		}
		resources = append(resources, sourceRBACProxy.Resources()...)
		resources = append(resources, r.rawKafkaSourceManifest.Resources()...)
	}

	// Kafka Broker Data Plane
	if build == manifestBuildAll || (build == manifestBuildEnabledOnly && instance.Spec.Broker.Enabled) || (build == manifestBuildDisabledOnly && !instance.Spec.Broker.Enabled) {
		rbacProxy, err := monitoring.AddRBACProxyToManifest(instance, monitoring.KafkaBrokerReceiver, monitoring.KafkaBrokerDispatcher)
		if err != nil {
			return nil, err
		}
		resources = append(resources, rbacProxy.Resources()...)
		resources = append(resources, r.rawKafkaBrokerManifest.Resources()...)
	}

	// Kafka Sink Data Plan
	if build == manifestBuildAll || (build == manifestBuildEnabledOnly && instance.Spec.Sink.Enabled) || (build == manifestBuildDisabledOnly && !instance.Spec.Sink.Enabled) {
		rbacProxy, err := monitoring.AddRBACProxyToManifest(instance, monitoring.KafkaSinkReceiver)
		if err != nil {
			return nil, err
		}
		resources = append(resources, rbacProxy.Resources()...)
		resources = append(resources, r.rawKafkaSinkManifest.Resources()...)
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

func enableControlPlaneManifest(spec serverlessoperatorv1alpha1.KnativeKafkaSpec) bool {
	return spec.Broker.Enabled || spec.Sink.Enabled || spec.Source.Enabled || spec.Channel.Enabled
}

// configureEventingKafka configures the new Knative Eventing components for Apache Kafka
func configureEventingKafka(spec serverlessoperatorv1alpha1.KnativeKafkaSpec) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// patch the deployment and enable the relevant controllers
		if u.GetKind() == "Deployment" && u.GetName() == "kafka-controller" {

			var disabledKafkaControllers = common.StringMap{
				brokerController:  "broker-controller,trigger-controller,namespaced-broker-controller,namespaced-trigger-controller",
				sinkController:    "sink-controller",
				sourceController:  "source-controller",
				channelController: "channel-controller",
			}

			var deployment = &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return err
			}

			if spec.Broker.Enabled {
				// broker is enabled, so we remove all of its controllers from the list of disabled controllers
				disabledKafkaControllers.Remove(brokerController)
			}
			if spec.Sink.Enabled {
				// sink is enabled, so we remove all of its controllers from the list of disabled controllers
				disabledKafkaControllers.Remove(sinkController)
			}
			if spec.Source.Enabled {
				// source is enabled, so we remove all of its controllers from the list of disabled controllers
				disabledKafkaControllers.Remove(sourceController)
			}
			if spec.Channel.Enabled {
				// channel is enabled, so we remove all of its controllers from the list of disabled controllers
				disabledKafkaControllers.Remove(channelController)
			}

			// render the actual argument, if any
			if len(disabledKafkaControllers) == 0 {
				deployment.Spec.Template.Spec.Containers[0].Args = nil
			} else {
				deployment.Spec.Template.Spec.Containers[0].Args = []string{"--disable-controllers=" + disabledKafkaControllers.StringValues()}
			}

			return scheme.Scheme.Convert(deployment, u, nil)
		}

		if u.GetKind() == "ConfigMap" && u.GetName() == "kafka-config-logging" {
			log.Info("Found ConfigMap kafka-config-logging, updating it with values from spec")

			if err := unstructured.SetNestedField(u.Object, renderLoggingConfigXML(spec.Logging), "data", "config.xml"); err != nil {
				return err
			}
		}

		// configure the broker itself
		if u.GetKind() == "ConfigMap" && u.GetName() == "kafka-broker-config" {
			log.Info("Found ConfigMap kafka-broker-config, updating it with values from spec")

			kafkaBrokerDefaultConfig := spec.Broker.DefaultConfig
			if err := unstructured.SetNestedField(u.Object, kafkaBrokerDefaultConfig.BootstrapServers, "data", "bootstrap.servers"); err != nil {
				return err
			}

			if err := unstructured.SetNestedField(u.Object, strconv.FormatInt(int64(kafkaBrokerDefaultConfig.NumPartitions), 10), "data", "default.topic.partitions"); err != nil {
				return err
			}

			if err := unstructured.SetNestedField(u.Object, strconv.FormatInt(int64(kafkaBrokerDefaultConfig.ReplicationFactor), 10), "data", "default.topic.replication.factor"); err != nil {
				return err
			}
			if kafkaBrokerDefaultConfig.AuthSecretName != "" {
				if err := unstructured.SetNestedField(u.Object, kafkaBrokerDefaultConfig.AuthSecretName, "data", "auth.secret.ref.name"); err != nil {
					return err
				}
			}
		}

		// configure the channel itself
		if u.GetKind() == "ConfigMap" && u.GetName() == "kafka-channel-config" {
			log.Info("Found ConfigMap kafka-channel-config, updating it with values from spec")

			kafkaChannelConfig := spec.Channel
			if err := unstructured.SetNestedField(u.Object, kafkaChannelConfig.BootstrapServers, "data", "bootstrap.servers"); err != nil {
				return err
			}

			if kafkaChannelConfig.AuthSecretName != "" {
				if err := unstructured.SetNestedField(u.Object, kafkaChannelConfig.AuthSecretName, "data", "auth.secret.ref.name"); err != nil {
					return err
				}
			}

			if kafkaChannelConfig.AuthSecretNamespace != "" {
				if err := unstructured.SetNestedField(u.Object, kafkaChannelConfig.AuthSecretNamespace, "data", "auth.secret.ref.namespace"); err != nil {
					return err
				}
			}

		}
		return nil
	}
}

func contains(array []string, name string) bool {
	for _, component := range array {
		if name == component {
			return true
		}
	}
	return false
}

func renderLoggingConfigXML(logging *serverlessoperatorv1alpha1.Logging) string {
	loglevel := "INFO"
	if logging != nil {
		loglevel = logging.Level
	}

	xmlTemplate := `    <configuration>
      <appender name="jsonConsoleAppender" class="ch.qos.logback.core.ConsoleAppender">
        <encoder class="net.logstash.logback.encoder.LogstashEncoder"/>
      </appender>
      <root level="` + loglevel + `">
        <appender-ref ref="jsonConsoleAppender"/>
      </root>
    </configuration>`
	return xmlTemplate
}

func setKafkaDeployments(replicas int32) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && contains(KafkaHAComponents, u.GetName()) {
			log.Info("Setting Kafka HA component", "deployment", u.GetName(), "replicas", replicas)
			if err := unstructured.SetNestedField(u.Object, int64(replicas), "spec", "replicas"); err != nil {
				return err
			}
		}
		return nil
	}
}

func executeStages(instance *serverlessoperatorv1alpha1.KnativeKafka, manifest *mf.Manifest, stages []stage) error {
	// Execute each stage in sequence until one returns an error
	for _, stage := range stages {
		if err := stage(manifest, instance); err != nil {
			return err
		}
	}
	return nil
}

func removeCreationTimestamp(manifest *mf.Manifest, _ *serverlessoperatorv1alpha1.KnativeKafka) error {
	// Avoid multiple unnecessary resource updates and reconciler looping
	// due to creationTimestamp set to `null` on some resources.
	var err error
	*manifest, err = manifest.Transform(func(u *unstructured.Unstructured) error {
		u.SetCreationTimestamp(metav1.Time{})
		return nil
	})
	return err
}
