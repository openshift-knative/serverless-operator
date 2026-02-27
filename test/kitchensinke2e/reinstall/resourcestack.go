package reinstall

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

/*
ResourceStack stores a stack of resources in Context.
The original use case is the re-install test, in which part of the installed resources are removed from a cluster
and then re-installed again later in the test.
*/
type ResourceStack struct {
	stack []*unstructured.Unstructured
	mu    sync.Mutex
}

func (f *ResourceStack) Push(unstructured *unstructured.Unstructured) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stack = append(f.stack, unstructured)
}

func (f *ResourceStack) Pop() *unstructured.Unstructured {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.stack) == 0 {
		return nil
	}
	ret := f.stack[len(f.stack)-1]
	f.stack = f.stack[:len(f.stack)-1]
	return ret
}

type resourceStackKey struct{}

func ContextWithResourceStack(ctx context.Context, store *ResourceStack) context.Context {
	return context.WithValue(ctx, resourceStackKey{}, store)
}

func ResourceStackFromContext(ctx context.Context) *ResourceStack {
	if e, ok := ctx.Value(resourceStackKey{}).(*ResourceStack); ok {
		return e
	}
	panic("no ResourceStack found in context")
}
