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

func TestTargetRouteDefaulting(t *testing.T) {
	tests := []struct {
		name string
		in   *TargetRoute
		want *TargetRoute
	}{{
		name: "empty",
		in:   &TargetRoute{},
		want: &TargetRoute{
			servingv1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						openshiftPassthrough: "true",
					},
				},
			},
		},
	}, {
		name: "override",
		in: &TargetRoute{
			servingv1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						openshiftPassthrough: "false",
					},
				},
			},
		},
		want: &TargetRoute{
			servingv1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						openshiftPassthrough: "true",
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

func TestValidateRoute(t *testing.T) {
	in := &TargetRoute{}

	if in.Validate(context.Background()) != nil {
		t.Error("Validate should have returned nil")
	}
}

func TestDeepCopyObjectRoute(t *testing.T) {

	tests := []struct {
		name string
		in   *TargetRoute
	}{{
		name: "with name",
		in: &TargetRoute{
			servingv1.Route{
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
