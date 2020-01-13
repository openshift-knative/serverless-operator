package util

import (
	"errors"
	"sync"
	"time"

	crdapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	crdinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// WaitForCRDs waits for the given crdTypes to appear as APIs.
func WaitForCRDs(mgr manager.Manager, stopCh <-chan struct{}, crdTypes ...runtime.Object) error {
	crdClient, err := crdclient.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	crdInformerFactory := crdinformers.NewSharedInformerFactory(crdClient, 10*time.Hour)
	crdInformer := crdInformerFactory.Apiextensions().V1beta1().CustomResourceDefinitions().Informer()

	// Produce the GVKs for the passed in types.
	scheme := mgr.GetScheme()
	wantedGvks := make(map[schema.GroupVersionKind]bool)
	for _, typ := range crdTypes {
		typeGvks, _, err := scheme.ObjectKinds(typ)
		if err != nil {
			return err
		}
		for _, gvk := range typeGvks {
			wantedGvks[gvk] = true
		}
	}

	// This channel will be closed once all types are available.
	doneCh := make(chan struct{})
	doneOnce := sync.Once{}
	handler := func(_ interface{}) {
		if allTypesAvailable(mgr, crdTypes...) {
			doneOnce.Do(func() {
				close(doneCh)
			})
		}
	}

	crdInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		// Filter by the GVKs we're actually looking for (from the passed in types) as
		// trying out all possible GVKs in the cluster takes ages.
		FilterFunc: func(obj interface{}) bool {
			crd := obj.(*crdapi.CustomResourceDefinition)
			for _, gvk := range possibleGvksFromCRD(crd) {
				if wantedGvks[gvk] {
					return true
				}
			}
			return false
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    handler,
			UpdateFunc: passNew(handler),
			DeleteFunc: handler,
		},
	})

	// Stop the informer once we finish here.
	innerStopCh := make(chan struct{})
	defer close(innerStopCh)
	go crdInformer.Run(innerStopCh)

	select {
	case <-doneCh:
	case <-stopCh:
		return errors.New("stopped before all types have been verified")
	}
	return nil
}

// passNew adapts a function taking only one object (the new one) to fit the update handler signature.
func passNew(f func(interface{})) func(interface{}, interface{}) {
	return func(_ interface{}, new interface{}) {
		f(new)
	}
}

// possibleGvksFromCRD produces a list of GroupVersionKinds that a given CRD represents.
func possibleGvksFromCRD(crd *crdapi.CustomResourceDefinition) []schema.GroupVersionKind {
	var gvks []schema.GroupVersionKind
	for _, version := range crd.Spec.Versions {
		gvk := schema.GroupVersionKind{
			Group:   crd.Spec.Group,
			Version: version.Name,
			Kind:    crd.Spec.Names.Kind,
		}
		gvks = append(gvks, gvk)
	}
	return gvks
}

// allTypesAvailable returns whether or not an informer for all the given types can
// be created without an error.
func allTypesAvailable(mgr manager.Manager, crdTypes ...runtime.Object) bool {
	for _, typ := range crdTypes {
		if _, err := mgr.GetCache().GetInformer(typ); err != nil {
			return false
		}
	}
	return true
}
