// Code generated by injection-gen. DO NOT EDIT.

package fake

import (
	context "context"

	console "github.com/openshift-knative/serverless-operator/pkg/client/config/injection/informers/config/v1/console"
	fake "github.com/openshift-knative/serverless-operator/pkg/client/config/injection/informers/factory/fake"
	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
)

var Get = console.Get

func init() {
	injection.Fake.RegisterInformer(withInformer)
}

func withInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := fake.Get(ctx)
	inf := f.Config().V1().Consoles()
	return context.WithValue(ctx, console.Key{}, inf), inf.Informer()
}
