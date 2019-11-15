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

package test

import "testing"

// Runner is a test runner that takes a context into consideration
type Runner interface {
	Run(name string, testfunc func(t *testing.T)) bool
}

func (spec Specification) run(ctx *Context) bool {
	tt := ctx.t
	if spec.contextual() {
		cspec := spec.contextSpec
		return tt.Run(cspec.name, func(t *testing.T) {
			ctx.t = t
			ctx.push(cspec.name)
			defer ctx.pop()
			cspec.testfunc(ctx)
		})
	} else {
		return tt.Run(spec.regularSpec.name, spec.regularSpec.testfunc)
	}
}
