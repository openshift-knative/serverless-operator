package common

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Set some data in a configmap, only overwriting common keys if they differ
func UpdateConfigMap(cm *unstructured.Unstructured, data map[string]string, log logr.Logger) {
	for k, v := range data {
		message := []interface{}{"map", cm.GetName(), k, v}
		if x, found, _ := unstructured.NestedFieldNoCopy(cm.Object, "data", k); found {
			if v == x {
				continue
			}
			message = append(message, "previous", x)
		}
		log.Info("Setting", message...)
		unstructured.SetNestedField(cm.Object, v, "data", k)
	}
}
