package main

import (
	"context"
	"fmt"
	"log"

	triggerutil "github.com/openshift/library-go/pkg/image/trigger"
	imagetriggercontroller "github.com/openshift/openshift-controller-manager/pkg/image/controller/trigger"
	triggerannotations "github.com/openshift/openshift-controller-manager/pkg/image/trigger/annotations"

	ocpclient "github.com/openshift-knative/serverless-operator/pkg/client/injection/client"
	imagestreaminformer "github.com/openshift-knative/serverless-operator/pkg/client/injection/informers/image/v1/imagestream"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingclientset "knative.dev/serving/pkg/client/clientset/versioned"
	servingclient "knative.dev/serving/pkg/client/injection/client"
	kserviceinformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/service"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/signals"
)

func main() {
	ctx := signals.NewContext()
	cfg := injection.ParseAndGetRESTConfigOrDie()

	log.Printf("Registering %d clients", len(injection.Default.GetClients()))
	log.Printf("Registering %d informer factories", len(injection.Default.GetInformerFactories()))
	log.Printf("Registering %d informers", len(injection.Default.GetInformers()))

	ctx, startInformers := injection.EnableInjectionOrDie(ctx, cfg)

	client := servingclient.Get(ctx)
	ksvcInformer := kserviceinformer.Get(ctx)

	ocpClient := ocpclient.Get(ctx)
	imageCfg, err := ocpClient.ConfigV1().Images().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Failed to fetch image configuration: %v", err)
	}
	log.Printf("Internal registry hostname: %q", imageCfg.Status.InternalRegistryHostname)

	updater := podSpecUpdater{client: client}
	broadcaster := imagetriggercontroller.NewTriggerEventBroadcaster(kubeclient.Get(ctx).CoreV1())

	sources := []imagetriggercontroller.TriggerSource{{
		Resource:  servingv1.SchemeGroupVersion.WithResource("services").GroupResource(),
		Informer:  ksvcInformer.Informer(),
		Store:     ksvcInformer.Informer().GetIndexer(),
		TriggerFn: triggerannotations.NewAnnotationTriggerIndexer,
		Reactor:   &triggerutil.AnnotationReactor{Updater: updater},
	}}

	startInformers()

	imagetriggercontroller.NewTriggerController(
		imageCfg.Status.InternalRegistryHostname,
		broadcaster,
		outerInformerWrapper{ours: imagestreaminformer.Get(ctx)},
		sources...,
	).Run(5, ctx.Done())
}

type podSpecUpdater struct {
	client servingclientset.Interface
}

func (u podSpecUpdater) Update(obj runtime.Object) error {
	switch t := obj.(type) {
	case *servingv1.Service:
		_, err := u.client.ServingV1().Services(t.Namespace).Update(context.TODO(), t, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("unrecognized object - no trigger update possible for %T", obj)
	}
}
