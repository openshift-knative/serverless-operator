package reinstall

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
)

var ServingV1beta1 = schema.GroupVersionResource{Group: "operator.knative.dev", Version: "v1beta1", Resource: "knativeservings"}
var EventingV1beta1 = schema.GroupVersionResource{Group: "operator.knative.dev", Version: "v1beta1", Resource: "knativeeventings"}
var KnativeKafkaV1alpha1 = schema.GroupVersionResource{Group: "operator.serverless.openshift.io", Version: "v1alpha1", Resource: "knativekafkas"}

var Serving = ServingV1beta1
var Eventing = EventingV1beta1
var KnativeKafka = KnativeKafkaV1alpha1

func uninstallResourceStep(gvr schema.GroupVersionResource, namespace string, name string) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		dynamicClient := dynamicclient.Get(ctx)
		resourceStack := ResourceStackFromContext(ctx)

		serving, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}

		resourceStack.Push(serving)

		err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func waitForPodNonExistence(namespace string, podLabelsToNotExist ...string) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		kube := kubeclient.Get(ctx)

		interval, timeout := k8s.PollTimings(ctx, nil)
		pods := kube.CoreV1().Pods(namespace)

		err := wait.PollImmediate(interval, timeout, func() (bool, error) {
			for _, labelSelector := range podLabelsToNotExist {
				plist, err := pods.List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
				if err != nil {
					return false, err
				}
				if len(plist.Items) != 0 {
					return false, nil
				}
			}
			return true, nil
		})

		if err != nil {
			t.Fatalf("Error waiting for pod non-existence in %s (%v): %v", namespace, podLabelsToNotExist, err)
		}
	}
}

func waitForKnativeReadiness(namespace string, name string, gvr schema.GroupVersionResource) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		err := k8s.WaitForResourceCondition(ctx, t, namespace, name, gvr, func(resource duckv1.KResource) bool {
			for _, condition := range resource.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == "True" {
					return true
				}
			}
			return false
		})
		if err != nil {
			t.Fatalf("Error waiting for Resource %s Readiness in %s (%v): %v", name, namespace, gvr, err)
		}
	}
}

func uninstallKnativeServingStep() feature.StepFn {
	return uninstallResourceStep(Serving, "knative-serving", "knative-serving")
}

func uninstallKnativeEventingStep() feature.StepFn {
	return uninstallResourceStep(Eventing, "knative-eventing", "knative-eventing")
}

func uninstallKnativeKafkaStep() feature.StepFn {
	return uninstallResourceStep(KnativeKafka, "knative-eventing", "knative-kafka")
}

func reinstallResources() feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		dynamicClient := dynamicclient.Get(ctx)
		resourceStack := ResourceStackFromContext(ctx)
		for resource := resourceStack.Pop(); resource != nil; resource = resourceStack.Pop() {
			gvr, _ := meta.UnsafeGuessKindToResource(resource.GroupVersionKind())

			// As we re-create the object, we clean up the Object metadata by creating a fresh one, copying the .spec
			resourceSpec, _, _ := unstructured.NestedMap(resource.Object, "spec")
			blankedUnstructured := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": resource.GroupVersionKind().GroupVersion().String(),
					"kind":       resource.GroupVersionKind().Kind,
					"metadata": map[string]interface{}{
						"name":        resource.GetName(),
						"labels":      resource.GetLabels(),
						"annotations": resource.GetAnnotations(),
					},
					"spec": resourceSpec,
				},
			}

			_, err := dynamicClient.Resource(gvr).Namespace(resource.GetNamespace()).Create(ctx, blankedUnstructured, metav1.CreateOptions{})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestUninstalledFeatureSet(ctx context.Context, env environment.Environment, t *testing.T, fss ...feature.FeatureSet) {
	t.Run("uninstall Serverless", func(t *testing.T) {
		uninstallAll := feature.NewFeatureNamed("Uninstall Serverless")
		// TODO: workaround for https://issues.redhat.com/browse/SRVKS-1325 , skip Serving reinstall for now
		//uninstallAll.Setup("Uninstall Serving", uninstallKnativeServingStep())
		//uninstallAll.Assert("Wait for Serving controllers non-existence", waitForPodNonExistence("knative-serving", "app.kubernetes.io/component=controller", "app.kubernetes.io/component=webhook"))

		uninstallAll.Setup("Uninstall Eventing and KnativeKafka", func(ctx context.Context, t feature.T) {
			// need to be sync, to be sure we install KnativeEventing first when re-installing
			uninstallKnativeKafkaStep()(ctx, t)
			uninstallKnativeEventingStep()(ctx, t)
		})
		uninstallAll.Assert("Wait for Eventing controllers non-existence", waitForPodNonExistence("knative-eventing", "app.kubernetes.io/component=eventing-controller", "app.kubernetes.io/component=eventing-webhook"))
		uninstallAll.Assert("Wait for Eventing Kafka controllers non-existence", waitForPodNonExistence("knative-eventing", "app.kubernetes.io/component=kafka-controller", "app.kubernetes.io/component=kafka-webhook-eventing"))

		env.Test(ctx, t, uninstallAll)
	})

	t.Run("setups", func(t *testing.T) {
		for _, fs := range fss {
			fs := fs
			for _, f := range fs.Features {
				f := f

				preInstallSteps := make([]feature.Step, 0, len(f.Steps))

				for _, s := range f.Steps {
					if s.T != feature.Assert && s.T != feature.Teardown {
						preInstallSteps = append(preInstallSteps, s)
					}
				}

				preInstallFeature := feature.NewFeatureNamed(f.Name + " pre-install steps")
				preInstallFeature.Steps = preInstallSteps

				// Run features within feature sets in parallel.
				t.Run(f.Name, func(t *testing.T) {
					t.Parallel()
					env.Test(ctx, t, preInstallFeature)
				})
			}
		}
	})

	t.Run("reinstall Serverless", func(t *testing.T) {
		reinstallServerless := feature.NewFeatureNamed("Reinstall Serverless")
		reinstallServerless.Setup("Reinstall Serverless", reinstallResources())
		reinstallServerless.Assert("Wait for KnativeServing Readiness", waitForKnativeReadiness("knative-serving", "knative-serving", Serving))
		reinstallServerless.Assert("Wait for KnativeEventing Readiness", waitForKnativeReadiness("knative-eventing", "knative-eventing", Eventing))
		reinstallServerless.Assert("Wait for KnativeKafka Readiness", waitForKnativeReadiness("knative-eventing", "knative-kafka", KnativeKafka))
		env.Test(ctx, t, reinstallServerless)
	})

	t.Run("assertions", func(t *testing.T) {
		for _, fs := range fss {
			fs := fs
			for _, f := range fs.Features {
				f := f

				postInstallSteps := make([]feature.Step, 0, len(f.Steps))

				for _, s := range f.Steps {
					if s.T == feature.Assert || s.T == feature.Teardown {
						postInstallSteps = append(postInstallSteps, s)
					}
				}

				postInstallFeature := feature.NewFeatureNamed(f.Name + " post-install steps")
				postInstallFeature.Steps = postInstallSteps

				// Run features within feature sets in parallel.
				t.Run(f.Name, func(t *testing.T) {
					t.Parallel()
					env.Test(ctx, t, postInstallFeature)
				})
			}
		}
	})
}
