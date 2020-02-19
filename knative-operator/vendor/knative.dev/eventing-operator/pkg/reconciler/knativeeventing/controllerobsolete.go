/*
Copyright 2020 The Knative Authors.
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

package knativeeventing

import (
	"context"

	eventinginformerObsolete "knative.dev/eventing-operator/pkg/client/injection/informers/eventing/v1alpha1/eventing"
	rbase "knative.dev/eventing-operator/pkg/reconciler"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

const (
	controllerAgentNameObsolete = "knativeeventing-controller-obsolete"
	reconcilerNameObsolete      = "KnativeEventing-obsolete"
)

// NewControllerObsolete initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewControllerObsolete(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	c := &ReconcilerObsolete{
		Base: rbase.NewBase(ctx, controllerAgentNameObsolete, cmw),
	}

	knativeEventingInformerObsolete := eventinginformerObsolete.Get(ctx)
	c.knativeEventingObsoleteLister = knativeEventingInformerObsolete.Lister()
	impl := controller.NewImpl(c, c.Logger, reconcilerNameObsolete)
	knativeEventingInformerObsolete.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	c.Logger.Info("Setting up event handlers for ", reconcilerNameObsolete)
	return impl
}
