package test

import "testing"

// Specification - describes a test with name and method that will be use as test
type Specification struct {
	Name string
	Func func(*testing.T)
}

// NewSpec - creates a new spec
func NewSpec(name string, lambda func(*testing.T)) Specification {
	return Specification{
		Name: name,
		Func: lambda,
	}
}
