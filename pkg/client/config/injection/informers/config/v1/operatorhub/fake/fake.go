// Code generated by injection-gen. DO NOT EDIT.

package fake

import (
	context "context"

	operatorhub "github.com/openshift-knative/serverless-operator/pkg/client/config/injection/informers/config/v1/operatorhub"
	fake "github.com/openshift-knative/serverless-operator/pkg/client/config/injection/informers/factory/fake"
	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
)

var Get = operatorhub.Get

func init() {
	injection.Fake.RegisterInformer(withInformer)
}

func withInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := fake.Get(ctx)
	inf := f.Config().V1().OperatorHubs()
	return context.WithValue(ctx, operatorhub.Key{}, inf), inf.Informer()
}
