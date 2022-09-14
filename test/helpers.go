package test

import (
	"context"
        "os"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func IsServiceMeshInstalled(ctx *Context) bool {
	_, err := ctx.Clients.Dynamic.Resource(schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}).Get(context.Background(), "servicemeshcontrolplanes.maistra.io", metav1.GetOptions{})

	if err == nil {
		return true
	}

	if !errors.IsNotFound(err) {
		ctx.T.Fatalf("Error checking if servicemeshcontrolplanes.maistra.io CRD exists: %v", err)
	}

	return false
}

func GetRegistryFromEnv() string {
	if value, ok := os.LookupEnv("IMAGE_REGISTRY_NAME"); ok {
		return value
	}
	return "quay.io"
}
