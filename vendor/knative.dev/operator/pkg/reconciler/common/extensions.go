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
package common

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

// Extension enables platform-specific features
type Extension interface {
	Transformers(v1alpha1.KComponent) []mf.Transformer
	Reconcile(context.Context, v1alpha1.KComponent) error
	Finalize(context.Context, v1alpha1.KComponent) error
}

// ExtensionGenerator creates an Extension from a Context
type ExtensionGenerator func(context.Context) Extension

// NoPlatform "generates" a NilExtension
func NoExtension(context.Context) Extension {
	return nilExtension{}
}

type nilExtension struct{}

func (nilExtension) Transformers(v1alpha1.KComponent) []mf.Transformer {
	return nil
}
func (nilExtension) Reconcile(context.Context, v1alpha1.KComponent) error {
	return nil
}
func (nilExtension) Finalize(context.Context, v1alpha1.KComponent) error {
	return nil
}
