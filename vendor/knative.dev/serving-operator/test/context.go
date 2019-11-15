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

// Context represents a testing context that can be tuned to override specific
// parts of a suite
type Context struct {
	t           *testing.T
	overrides   []Specification
	activeTests []string
}

// T returns standard testing.T
func (ctx Context) T() *testing.T {
	return ctx.t
}

// Runner creates a new runner based on a given context
func (ctx Context) Runner() Runner {
	return contextRunner{
		ctx: &ctx,
	}
}

// RunSuite executes given test or execute it's override
func (runner contextRunner) Run(name string, testfunc func(t *testing.T)) bool {
	ctx := runner.ctx
	t := ctx.t
	for _, spec := range ctx.overrides {
		if spec.matchesNameInContext(name, ctx) {
			t.Logf("Overriding %s test", spec.name())
			return t.Run(name, spec.testfunc(ctx))
		}
	}
	return t.Run(name, testfunc)
}

// WithOverride adds specification to be executed instead of given test
func (ctx *Context) WithOverride(spec Specification) *Context {
	ctx.overrides = append(ctx.overrides, spec)
	return ctx
}

// RunSuite will run a test suite within given context
func (ctx *Context) RunSuite(suite []Specification) {
	for _, spec := range suite {
		spec.run(ctx)
	}
}

// NewContext creates a new context
func NewContext(t *testing.T) *Context {
	return &Context{
		t:           t,
		overrides:   make([]Specification, 0),
		activeTests: make([]string, 0),
	}
}

func (ctx *Context) push(testname string) {
	ctx.activeTests = append(ctx.activeTests, testname)
}

func (ctx *Context) pop() string {
	// Top element
	n := len(ctx.activeTests) - 1
	testname := ctx.activeTests[n]
	ctx.activeTests = ctx.activeTests[:n]
	return testname
}

type contextRunner struct {
	ctx *Context
}
