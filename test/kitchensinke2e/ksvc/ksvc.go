package ksvc

import (
	"context"
	"embed"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/manifest"
	"github.com/openshift-knative/serverless-operator/test"
)

//go:embed *.yaml
var yaml embed.FS

var (
        defaultImage = test.GetRegistryFromEnv() + "/openshift-knative/helloworld-go:multiarch"
)

func GVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "serving.knative.dev", Version: "v1", Resource: "services"}
}

// Install will create a knative Service resource, using the latest version, augmented with the config fn options.
func Install(name string, opts ...manifest.CfgFn) feature.StepFn {
	cfg := map[string]interface{}{
		"name":    name,
		"version": GVR().Version,
		"image":   defaultImage,
	}
	for _, fn := range opts {
		fn(cfg)
	}
	return func(ctx context.Context, t feature.T) {
		if _, err := manifest.InstallYamlFS(ctx, yaml, cfg); err != nil {
			t.Fatal(err)
		}
	}
}

// IsReady tests to see if a knative Service becomes ready within the time given.
func IsReady(name string, timings ...time.Duration) feature.StepFn {
	return k8s.IsReady(GVR(), name, timings...)
}

func WithVersion(version string) manifest.CfgFn {
	return func(cfg map[string]interface{}) {
		if version != "" {
			cfg["version"] = version
		}
	}
}

func WithImage(image string) manifest.CfgFn {
	return func(cfg map[string]interface{}) {
		if image != "" {
			cfg["image"] = image
		}
	}
}

func WithEnv(name, value string) manifest.CfgFn {
	return func(cfg map[string]interface{}) {
		env, ok := cfg["env"]
		if !ok {
			env = make([]corev1.EnvVar, 0, 1)
		}

		envTyped := env.([]corev1.EnvVar)

		envTyped = append(envTyped, corev1.EnvVar{Name: name, Value: value})
		cfg["env"] = envTyped
	}
}
