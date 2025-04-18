/*
Copyright 2020 The Knative Authors

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

package common

import (
	"fmt"

	mf "github.com/manifestival/manifestival"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

const istioLabelName = "sidecar.istio.io/inject"

// JobTransform updates the job with the expected value for the key app in the label
func JobTransform(obj base.KComponent) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Job" {
			job := &batchv1.Job{}
			if err := scheme.Scheme.Convert(u, job, nil); err != nil {
				return err
			}

			component := "serving"
			if _, ok := obj.(*v1beta1.KnativeEventing); ok {
				component = ""
			}
			if job.GetName() == "" {
				job.SetName(fmt.Sprintf("%s%s-%s", job.GetGenerateName(), component, TargetVersion(obj)))
			} else {
				job.SetName(fmt.Sprintf("%s-%s-%s", job.GetName(), component, TargetVersion(obj)))
			}

			addIstioIgnoreLabels(job)
			return scheme.Scheme.Convert(job, u, nil)
		}

		return nil
	}
}

func addIstioIgnoreLabels(job *batchv1.Job) {
	labels := job.Spec.Template.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	istioLabel := labels[istioLabelName]
	if istioLabel == "" {
		labels[istioLabelName] = "false"
		job.Spec.Template.SetLabels(labels)
	}
}
