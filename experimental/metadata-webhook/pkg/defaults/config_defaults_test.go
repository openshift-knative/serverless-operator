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

package defaults

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/google/go-cmp/cmp"
)

func TestTargetConfigurationDefaulting(t *testing.T) {
	tests := []struct {
		name string
		in   *TargetConfiguration
		want *TargetConfiguration
	}{{
		name: "empty",
		in:   &TargetConfiguration{},
		want: &TargetConfiguration{
			servingv1.Configuration{
				Spec: servingv1.ConfigurationSpec{
					Template: servingv1.RevisionTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								sidecarInject:                "true",
								sidecarrewriteAppHTTPProbers: "true",
							},
						},
					},
				},
			},
		},
	}, {
		name: "override",
		in: &TargetConfiguration{
			servingv1.Configuration{
				Spec: servingv1.ConfigurationSpec{
					Template: servingv1.RevisionTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								sidecarInject:                "false",
								sidecarrewriteAppHTTPProbers: "false",
							},
						},
					},
				},
			},
		},
		want: &TargetConfiguration{
			servingv1.Configuration{
				Spec: servingv1.ConfigurationSpec{
					Template: servingv1.RevisionTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								sidecarInject:                "true",
								sidecarrewriteAppHTTPProbers: "true",
							},
						},
					},
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.in
			got.SetDefaults(context.Background())
			if !cmp.Equal(got, test.want) {
				t.Errorf("SetDefaults (-want, +got) = %v",
					cmp.Diff(test.want, got))
			}
		})
	}
}

func TestValidateConfiguration(t *testing.T) {
	in := &TargetConfiguration{}

	if in.Validate(context.Background()) != nil {
		t.Error("Validate should have returned nil")
	}
}

func TestDeepCopyObjectConfiguration(t *testing.T) {

	tests := []struct {
		name string
		in   *TargetConfiguration
	}{{
		name: "with name",
		in: &TargetConfiguration{
			servingv1.Configuration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo-deployment",
				},
			},
		},
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := test.in

			got := in.DeepCopyObject()

			if got == in {
				t.Error("DeepCopyInto returned same object")
			}

			if !cmp.Equal(in, got) {
				t.Errorf("DeepCopyInto (-in, +got) = %v",
					cmp.Diff(in, got))
			}
		})
	}
}
