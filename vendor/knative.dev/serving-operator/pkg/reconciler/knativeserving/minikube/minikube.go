/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package minikube

import (
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	mf "github.com/jcrossley3/manifestival"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/serving-operator/pkg/reconciler/knativeserving/common"
)

var log *zap.SugaredLogger

// Configure minikube if we're soaking in it
func Configure(kubeClientSet kubernetes.Interface, slog *zap.SugaredLogger) (mf.Transformer, error) {
	log = slog.Named("minikube")
	if _, err := kubeClientSet.CoreV1().Nodes().Get("minikube", metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Unable to query for minikube node")
		}
		// Not running in minikube
		return nil, nil
	}
	return egress, nil
}

func egress(u *unstructured.Unstructured) error {
	if u.GetKind() == "ConfigMap" && u.GetName() == "config-network" {
		data := map[string]string{
			"istio.sidecar.includeOutboundIPRanges": "10.0.0.1/24",
		}
		common.UpdateConfigMap(u, data, log)
	}
	return nil
}
