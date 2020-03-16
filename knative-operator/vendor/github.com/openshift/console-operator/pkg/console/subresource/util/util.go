package util

import (
	"fmt"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/console-operator/pkg/api"
)

func SharedLabels() map[string]string {
	return map[string]string{
		"app": api.OpenShiftConsoleName,
	}
}

func LabelsForConsole() map[string]string {
	baseLabels := SharedLabels()

	extraLabels := map[string]string{
		"component": "ui",
	}
	// we want to deduplicate, so doing these two loops.
	allLabels := map[string]string{}

	for key, value := range baseLabels {
		allLabels[key] = value
	}
	for key, value := range extraLabels {
		allLabels[key] = value
	}
	return allLabels
}

func SharedMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        api.OpenShiftConsoleName,
		Namespace:   api.OpenShiftConsoleNamespace,
		Labels:      SharedLabels(),
		Annotations: map[string]string{},
	}
}

func LogYaml(obj runtime.Object) {
	// REALLY NOISY, but handy for debugging:
	// deployJSON, err := json.Marshal(d)
	objYaml, err := yaml.Marshal(obj)
	if err != nil {
		klog.V(4).Infoln("failed to show yaml in log")
	} else {
		klog.V(4).Infof("%v", string(objYaml))
	}
}

// objects can have more than one ownerRef, potentially
func AddOwnerRef(obj metav1.Object, ownerRef *metav1.OwnerReference) {
	// TODO: find the library-go equivalent of this and replace
	// currently errs out with something like:
	// failed with: ConfigMap "console-config" is invalid: [metadata.ownerReferences.apiVersion: Invalid value: "": version must not be empty, metadata.ownerReferences.kind: Invalid value: "": kind must not be empty]
	//if obj != nil {
	//	if ownerRef != nil {
	//		obj.SetOwnerReferences(append(obj.GetOwnerReferences(), *ownerRef))
	//	}
	//}
}

// func RemoveOwnerRef
func OwnerRefFrom(cr *operatorv1.Console) *metav1.OwnerReference {

	if cr != nil {
		truthy := true
		return &metav1.OwnerReference{
			APIVersion: cr.APIVersion,
			Kind:       cr.Kind,
			Name:       cr.Name,
			UID:        cr.UID,
			Controller: &truthy,
		}
	}
	return nil
}

// borrowed from library-go
// https://github.com/openshift/library-go/blob/master/pkg/operator/v1alpha1helpers/helpers.go
func GetImageEnv() string {
	return os.Getenv("IMAGE")
}

// TODO: technically, this should take targetPort from route.spec.port.targetPort
func HTTPS(host string) string {
	protocol := "https://"
	if host == "" {
		klog.V(4).Infoln("util.HTTPS() cannot accept an empty string.")
		return ""
	}
	if strings.HasPrefix(host, protocol) {
		return host
	}
	secured := fmt.Sprintf("%s%s", protocol, host)
	return secured
}
