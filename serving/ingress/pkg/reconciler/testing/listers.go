package testing

import (
	routev1 "github.com/openshift/api/route/v1"
	fakerouteclientset "github.com/openshift/client-go/route/clientset/versioned/fake"
	routev1listers "github.com/openshift/client-go/route/listers/route/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	networking "knative.dev/networking/pkg/apis/networking/v1alpha1"
	fakenetworkingclientset "knative.dev/networking/pkg/client/clientset/versioned/fake"
	networkinglisters "knative.dev/networking/pkg/client/listers/networking/v1alpha1"
	"knative.dev/pkg/reconciler/testing"
)

var clientSetSchemes = []func(*runtime.Scheme) error{
	fakenetworkingclientset.AddToScheme,
	fakerouteclientset.AddToScheme,
}

type Listers struct {
	sorter testing.ObjectSorter
}

func NewListers(objs []runtime.Object) Listers {
	scheme := NewScheme()

	ls := Listers{
		sorter: testing.NewObjectSorter(scheme),
	}

	ls.sorter.AddObjects(objs...)

	return ls
}

func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	for _, addTo := range clientSetSchemes {
		addTo(scheme)
	}
	return scheme
}

func (*Listers) NewScheme() *runtime.Scheme {
	return NewScheme()
}

// IndexerFor returns the indexer for the given object.
func (l *Listers) IndexerFor(obj runtime.Object) cache.Indexer {
	return l.sorter.IndexerForObjectType(obj)
}

func (l *Listers) GetNetworkingObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakenetworkingclientset.AddToScheme)
}

func (l *Listers) GetRouteObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakerouteclientset.AddToScheme)
}

// GetIngressLister get lister for Ingress resource.
func (l *Listers) GetIngressLister() networkinglisters.IngressLister {
	return networkinglisters.NewIngressLister(l.IndexerFor(&networking.Ingress{}))
}

// GetRouteLister get lister for Route resource.
func (l *Listers) GetRouteLister() routev1listers.RouteLister {
	return routev1listers.NewRouteLister(l.IndexerFor(&routev1.Route{}))
}
