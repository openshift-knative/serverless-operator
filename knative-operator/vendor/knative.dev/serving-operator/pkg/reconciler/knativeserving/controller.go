/*
Copyright 2019 The Knative Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package knativeserving

import (
	"context"
	"flag"
	"knative.dev/pkg/injection/sharedmain"
	"os"
	"path/filepath"

	"github.com/go-logr/zapr"
	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/cache"
	deploymentinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	knativeServinginformer "knative.dev/serving-operator/pkg/client/injection/informers/serving/v1alpha1/knativeserving"
	rbase "knative.dev/serving-operator/pkg/reconciler"
)

const (
	controllerAgentName = "knativeserving-controller"
	reconcilerName      = "KnativeServing"
)

var (
	recursive  = flag.Bool("recursive", false, "If filename is a directory, process all manifests recursively")
	MasterURL  = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	Kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	knativeServingInformer := knativeServinginformer.Get(ctx)
	deploymentInformer := deploymentinformer.Get(ctx)

	c := &Reconciler{
		Base:                 rbase.NewBase(ctx, controllerAgentName, cmw),
		knativeServingLister: knativeServingInformer.Lister(),
		servings:             map[string]int64{},
	}

	koDataDir := os.Getenv("KO_DATA_PATH")

	cfg, err := sharedmain.GetConfig(*MasterURL, *Kubeconfig)
	if err != nil {
		c.Logger.Error(err, "Error building kubeconfig")
	}

	mf.SetLogger(zapr.NewLogger(zap.NewExample()))
	config, err := mf.NewManifest(filepath.Join(koDataDir, "knative-serving/"), *recursive, cfg)
	if err != nil {
		c.Logger.Error(err, "Error creating the Manifest for knative-serving")
		os.Exit(1)
	}

	c.config = config
	impl := controller.NewImpl(c, c.Logger, reconcilerName)

	c.Logger.Info("Setting up event handlers for %s", reconcilerName)

	knativeServingInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("KnativeServing")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	return impl
}
