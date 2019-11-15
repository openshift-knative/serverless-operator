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

import (
	"strings"
	"testing"
)

// Specification describes a test with name and method that will be use as test
type Specification struct {
	regularSpec *regularSpecification
	contextSpec *contextualSpecification
}

// NewSpec creates a new test specification
func NewSpec(name string, testfunc func(t *testing.T)) Specification {
	return Specification{
		regularSpec: &regularSpecification{
			name:     name,
			testfunc: testfunc,
		},
		contextSpec: nil,
	}
}

// NewContextualSpec creates a new test specification that uses context
func NewContextualSpec(name string, testfunc func(ctx *Context)) Specification {
	return Specification{
		regularSpec: nil,
		contextSpec: &contextualSpecification{
			name:     name,
			testfunc: testfunc,
		},
	}
}

// Skip will skip a test by it's composite name
func Skip(testname string, args ...interface{}) Specification {
	return NewSpec(testname, func(t *testing.T) {
		t.Skip(args)
	})
}

// Skipf will skip a test by it's composite name
func Skipf(testname string, format string, args ...interface{}) Specification {
	return NewSpec(testname, func(t *testing.T) {
		t.Skipf(format, args)
	})
}

type regularSpecification struct {
	name     string
	testfunc func(t *testing.T)
}

type contextualSpecification struct {
	name     string
	testfunc func(ctx *Context)
}

func (spec Specification) contextual() bool {
	return spec.regularSpec == nil && spec.contextSpec != nil
}

func (spec Specification) name() string {
	if spec.contextual() {
		return spec.contextSpec.name
	} else {
		return spec.regularSpec.name
	}
}

func (spec Specification) matchesNameInContext(name string, ctx *Context) bool {
	candidate := strings.Join(ctx.activeTests[:], "/") + "/" + name
	return spec.name() == candidate
}

func (spec Specification) testfunc(ctx *Context) func(t *testing.T) {
	if spec.contextual() {
		return func(t *testing.T) {
			spec.contextSpec.testfunc(ctx)
		}
	}
	return spec.regularSpec.testfunc
}
