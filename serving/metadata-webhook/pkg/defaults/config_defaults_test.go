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
							Labels: map[string]string{
								sidecarInject: "true",
							},
							Annotations: map[string]string{
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
							Labels: map[string]string{
								sidecarInject: "false",
							},
							Annotations: map[string]string{
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
							Labels: map[string]string{
								sidecarInject: "true",
							},
							Annotations: map[string]string{
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
