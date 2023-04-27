package knativeservice

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"time"
)

func GVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "serving.knative.dev", Version: "v1", Resource: "services"}
}

// IsReady tests to see if a knative Service becomes ready within the time given.
func IsReady(name string, timings ...time.Duration) feature.StepFn {
	return k8s.IsReady(GVR(), name, timings...)
}
